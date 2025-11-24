package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/api"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/compare"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/worker"
)

type OrderService struct {
	client     *api.DropiClient
	workerPool *worker.WorkerPool
}

func NewOrderService(client *api.DropiClient, pool *worker.WorkerPool) *OrderService {
	return &OrderService{
		client:     client,
		workerPool: pool,
	}
}

type ProcessResult struct {
	TotalOrders      int           `json:"total_orders"`
	OrdersProcessed  int           `json:"orders_processed"`
	ChangesDetected  int           `json:"changes_detected"`
	WebhooksQueued   int           `json:"webhooks_queued"`   // Webhooks encolados
	WebhooksPending  int           `json:"webhooks_pending"`  // Webhooks aún procesándose
	OrdersSkipped    int           `json:"orders_skipped"`
	Errors           []string      `json:"errors,omitempty"`
	Details          []OrderStatus `json:"details"`
	PartialTimeout   bool          `json:"partial_timeout,omitempty"` // Indica si hubo timeout parcial
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
) (*ProcessResult, error) {

	const resultNumber = 50 // máximo permitido

	logger := slog.With(
		"country_suffix", countrySuffix,
		"webhook_suffix", webhookSuffix,
	)

	// 1) Consultar Dropi con paginación automática
	// Esto maneja automáticamente el caso de más de 50 órdenes
	orders, err := s.client.FetchAllOrders(ctx, apiKey, date, countrySuffix)
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

		// 3) Si cambió → Encolar webhook (ACTUALIZADO)
		if compareResult.Changed {
			result.ChangesDetected++

			s.workerPool.Enqueue(*order, webhookSuffix)
			result.WebhooksQueued++
			result.WebhooksPending++
		}
	}

	return result, nil
}
