package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/api"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/compare"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/webhook"
	"go.uber.org/zap"
)

type OrderService struct {
	client         *api.DropiClient
	webhookSender  *webhook.Sender
	maxConcurrency int // Máximo de webhooks concurrentes
}

func NewOrderService(client *api.DropiClient, sender *webhook.Sender) *OrderService {
	return &OrderService{
		client:         client,
		webhookSender:  sender,
		maxConcurrency: 5, // Reducido a 5 para evitar rate limiting (429)
	}
}

type ProcessResult struct {
	TotalOrders     int           `json:"total_orders"`
	OrdersProcessed int           `json:"orders_processed"`
	ChangesDetected int           `json:"changes_detected"`
	WebhooksQueued  int           `json:"webhooks_queued"`
	WebhooksPending int           `json:"webhooks_pending"`
	OrdersSkipped   int           `json:"orders_skipped"`
	Errors          []string      `json:"errors,omitempty"`
	Details         []OrderStatus `json:"details"`
	PartialTimeout  bool          `json:"partial_timeout,omitempty"`
}

type OrderStatus struct {
	OrderID       string   `json:"order_id"`
	ProductNames  []string `json:"product_names"`
	PreviousState string   `json:"previous_state"`
	CurrentState  string   `json:"current_state"`
	Changed       bool     `json:"changed"`
}

// ---------------------------------------------------------
// MÉTODO PRINCIPAL (Actualizado)
// ---------------------------------------------------------
func (s *OrderService) HandleOrderRequest(
	ctx context.Context,
	apiKey string,
	date string,
	countrySuffix string,
	webhookSuffix string,
	dateUtil string,
) (*ProcessResult, error) {

	logger := slog.With(
		"country_suffix", countrySuffix,
		"webhook_suffix", webhookSuffix,
	)

	// 1) Consultar Dropi con paginación automática
	// Esto maneja automáticamente el caso de más de 50 órdenes
	orders, err := s.client.FetchAllOrders(ctx, apiKey, date, countrySuffix, dateUtil)
	if err != nil {
		logger.Error("error fetching orders", "error", err)
		return nil, err
	}

	logger.Info("orders fetched from Dropi",
		"total_orders", len(orders),
		"date", date,
	)

	// Inicializar resultado
	result := &ProcessResult{
		TotalOrders: len(orders),
		Details:     make([]OrderStatus, 0),
		Errors:      []string{},
	}

	// Si no hay órdenes, retornar resultado vacío (no es un error)
	if len(orders) == 0 {
		logger.Info("no orders found for this date", "date", date)
		return result, nil
	}

	// Preparar para procesamiento concurrente de webhooks
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.maxConcurrency)
	webhookErrors := make(chan error, len(orders))

	for i := range orders {
		// Verificar si el context expiró antes de procesar cada orden
		select {
		case <-ctx.Done():
			// Timeout parcial: retornar resultado parcial sin error
			// Esto permite que el cliente vea las órdenes procesadas hasta el momento
			result.Errors = append(result.Errors, fmt.Sprintf("processing cancelled after %d orders: timeout", result.OrdersProcessed))
			result.PartialTimeout = true
			logger.Warn("partial timeout occurred",
				"orders_processed", result.OrdersProcessed,
				"orders_remaining", len(orders)-i,
			)
			return result, nil // Retornar nil en vez de ctx.Err() para evitar 500
		default:
			// Continuar
		}

		order := &orders[i]
		result.OrdersProcessed++

		// 2) Comparar estados
		compareResult, err := compare.CompareOrderStatus(order, logger)
		if err != nil {
			result.OrdersSkipped++
			errMsg := fmt.Sprintf("Order %d: %s", order.ID, err.Error())
			result.Errors = append(result.Errors, errMsg)

			logger.Warn("order skipped (cannot compare status)",
				"order_id", order.ID,
				"error", err,
			)
			continue
		}

		// Guardar detalle
		statusInfo := OrderStatus{
			OrderID:       fmt.Sprintf("%d", compareResult.OrderID),
			ProductNames:  compareResult.ProductNames,
			PreviousState: compareResult.OldStatus,
			CurrentState:  compareResult.NewStatus,
			Changed:       compareResult.Changed,
		}
		result.Details = append(result.Details, statusInfo)

		// 3) Si cambió → Enviar webhook de forma concurrente
		if compareResult.Changed {
			result.ChangesDetected++
			result.WebhooksQueued++

			wg.Add(1)
			go func(order models.DropiOrder, suffix string, orderID int64) {
				defer wg.Done()

				// Adquirir semáforo (limita concurrencia)
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				zap.L().Info("sending webhook",
					zap.Int64("order_id", orderID),
					zap.String("webhook_suffix", suffix),
				)

				if err := s.webhookSender.SendWebhook(order, suffix); err != nil {
					zap.L().Error("webhook failed",
						zap.Int64("order_id", orderID),
						zap.Error(err),
					)
					webhookErrors <- fmt.Errorf("order %d: %w", orderID, err)
				} else {
					zap.L().Info("webhook sent successfully",
						zap.Int64("order_id", orderID),
					)
				}
			}(*order, webhookSuffix, order.ID)
		}
	}

	// Esperar a que todos los webhooks terminen
	if result.WebhooksQueued > 0 {
		zap.L().Info("waiting for webhooks to complete",
			zap.Int("webhooks_queued", result.WebhooksQueued),
			zap.Int("max_concurrency", s.maxConcurrency),
		)

		// Log warning si hay muchos webhooks (caso pesado)
		if result.WebhooksQueued > 100 {
			zap.L().Warn("heavy load: processing many webhooks",
				zap.Int("webhook_count", result.WebhooksQueued),
				zap.Int("estimated_time_seconds", (result.WebhooksQueued/s.maxConcurrency)*3),
			)
		}

		wg.Wait()

		zap.L().Info("all webhooks completed",
			zap.Int("total_webhooks", result.WebhooksQueued),
		)
	}
	close(webhookErrors)

	// Recolectar errores de webhooks
	webhookErrorCount := 0
	for err := range webhookErrors {
		webhookErrorCount++
		result.Errors = append(result.Errors, err.Error())
	}

	if webhookErrorCount > 0 {
		zap.L().Warn("some webhooks failed",
			zap.Int("failed_count", webhookErrorCount),
			zap.Int("total_webhooks", result.WebhooksQueued),
		)
	}

	result.WebhooksPending = 0 // Todos completados

	return result, nil
}
