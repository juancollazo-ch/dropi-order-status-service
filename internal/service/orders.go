package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/api"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/compare"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/logging"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/models/serviceresponse"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/webhook"
	"go.uber.org/zap"
)

type OrderService struct {
	client         *api.DropiClient
	webhookSender  *webhook.Sender
	maxConcurrency int // Máximo de webhooks concurrentes
}

func NewOrderService(
	client *api.DropiClient,
	sender *webhook.Sender,
	maxConcurrency int,

) *OrderService {

	return &OrderService{
		client:         client,
		webhookSender:  sender,
		maxConcurrency: maxConcurrency,
	}
}

// MÉTODO PRINCIPAL (Actualizado)
func (s *OrderService) HandleOrderRequest(
	ctx context.Context,
	apiKey string,
	date string,
	countrySuffix string,
	webhookSuffix string,
	dateUtil string,
) (*serviceresponse.ProcessResult, error) {
	logFields := logging.GetLoggingFieldsFromContext(ctx)

	// 1) Consultar Dropi con paginación automática
	// Esto maneja automáticamente el caso de más de 50 órdenes
	orders, err := s.client.FetchAllOrders(ctx, apiKey, date, countrySuffix, dateUtil)

	if err != nil {
		// Log y propagar el error, ya que FetchAllOrders debería devolver un AppError
		zap.L().Error("error fetching orders from Dropi API",
			append(logFields,
				zap.Error(err),
				zap.String("country_suffix", countrySuffix),
				zap.String("webhook_suffix", webhookSuffix),
			)...,
		)
		return nil, err // Propagar el AppError de la capa API
	}

	zap.L().Info("orders fetched from Dropi",
		append(logFields,
			zap.Int("total_orders", len(orders)),
			zap.String("date", date),
			zap.String("country_suffix", countrySuffix),
		)...,
	)

	// Inicializar resultado
	result := &serviceresponse.ProcessResult{
		TotalOrders:    len(orders),
		Details:        make([]serviceresponse.OrderStatus, 0),
		FailedWebhooks: make([]serviceresponse.FailedWebhook, 0),
		Errors:         make([]string, 0),
	}

	// Si no hay órdenes, retornar resultado vacío (no es un error)
	if len(orders) == 0 {
		zap.L().Info("no orders found for this date",
			append(logFields,
				zap.String("date", date),
				zap.String("country_suffix", countrySuffix),
			)...,
		)
		return result, nil
	}

	// Preparar para procesamiento concurrente de webhooks
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.maxConcurrency)
	failedWebhooks := make(chan serviceresponse.FailedWebhook, len(orders))

	for i := range orders {
		// Verificar si el context expiró antes de procesar cada orden
		select {
		case <-ctx.Done():
			// Timeout parcial: retornar resultado parcial sin error
			// Esto permite que el cliente vea las órdenes procesadas hasta el momento
			result.Errors = append(result.Errors, fmt.Sprintf("processing cancelled after %d orders: timeout", result.OrdersProcessed))
			result.PartialTimeout = true
			zap.L().Warn("partial timeout occurred",
				append(logFields,
					zap.Int("orders_processed", result.OrdersProcessed),
					zap.Int("orders_remaining", len(orders)-i),
					zap.Error(ctx.Err()),
				)...,
			)
			return result, nil // Retornar nil en vez de ctx.Err() para evitar 500
		default:

		}

		order := &orders[i]
		result.OrdersProcessed++

		// 2) Comparar estados
		compareResult, err := compare.CompareOrderStatus(order)
		if err != nil {
			result.OrdersSkipped++
			errMsg := fmt.Sprintf("Order %d: %s", order.ID, err.Error())
			result.Errors = append(result.Errors, errMsg)

			zap.L().Warn("order skipped (cannot compare status)",
				append(logFields,
					zap.Int64("order_id", order.ID),
					zap.Error(err),
				)...,
			)
			continue
		}

		// Guardar detalle
		statusInfo := serviceresponse.OrderStatus{
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
			go func(order models.DropiOrder, suffix string, orderID int64, productNames []string) {
				defer wg.Done()

				// Adquirir semáforo (limita concurrencia)
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// El log detallado se hace en sender.go, aquí solo manejamos errores
				webhookURL, err := s.webhookSender.BuildWebhookURL(suffix)
				if err != nil {
					webhookURL = "invalid_url"
				}

				if err := s.webhookSender.SendWebhook(ctx, order, suffix); err != nil {
					zap.L().Error("webhook failed",
						append(logFields, // Incluir campos de logging aquí
							zap.Int64("order_id", orderID),
							zap.Error(err),
						)...,
					)

					// Agregar webhook fallido a la lista
					failedWebhooks <- serviceresponse.FailedWebhook{
						OrderID:      fmt.Sprintf("%d", orderID),
						ProductNames: productNames,
						Status:       order.Status,
						WebhookURL:   webhookURL,
						Error:        err.Error(),
					}
				}
			}(*order, webhookSuffix, order.ID, compareResult.ProductNames)
		}
	}

	// Esperar a que todos los webhooks terminen
	if result.WebhooksQueued > 0 {
		zap.L().Info("waiting for webhooks to complete",
			append(logFields, // Incluir campos de logging aquí
				zap.Int("webhooks_queued", result.WebhooksQueued),
				zap.Int("max_concurrency", s.maxConcurrency),
			)...,
		)

		// Log warning si hay muchos webhooks (caso pesado)
		if result.WebhooksQueued > 100 {
			zap.L().Warn("heavy load: processing many webhooks",
				append(logFields, // Incluir campos de logging aquí
					zap.Int("webhook_count", result.WebhooksQueued),
					zap.Int("estimated_time_seconds", (result.WebhooksQueued/s.maxConcurrency)*3),
				)...,
			)
		}

		wg.Wait()

		zap.L().Info("all webhooks completed",
			append(logFields, // Incluir campos de logging aquí
				zap.Int("total_webhooks", result.WebhooksQueued),
			)...,
		)
	}

	close(failedWebhooks)

	// Recolectar webhooks fallidos
	for failed := range failedWebhooks {
		result.FailedWebhooks = append(result.FailedWebhooks, failed)
		result.Errors = append(result.Errors, fmt.Sprintf("webhook for order %s failed: %s", failed.OrderID, failed.Error))
	}

	if len(result.FailedWebhooks) > 0 {
		zap.L().Warn("some webhooks failed",
			append(logFields,
				zap.Int("failed_count", len(result.FailedWebhooks)),
				zap.Int("total_webhooks", result.WebhooksQueued),
			)...,
		)
	}

	result.WebhooksPending = 0

	return result, nil
}
