package service

import (
	"context"
	"errors"
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
	TotalOrders     int           `json:"total_orders"`
	OrdersProcessed int           `json:"orders_processed"`
	ChangesDetected int           `json:"changes_detected"`
	WebhooksSent    int           `json:"webhooks_sent"`
	OrdersSkipped   int           `json:"orders_skipped"`
	Errors          []string      `json:"errors,omitempty"`
	Details         []OrderStatus `json:"details"`
}

type OrderStatus struct {
	OrderID       string `json:"order_id"`
	PreviousState string `json:"previous_state"`
	CurrentState  string `json:"current_state"`
	Changed       bool   `json:"changed"`
}

// ---------------------------------------------
// MÉTODO PRINCIPAL
// ---------------------------------------------
func (s *OrderService) HandleOrderRequest(ctx context.Context, apiKey, date string) (*ProcessResult, error) {

	const resultNumber = 50 // máximo de resultados según requerimiento

	// 1) Consultar Dropi
	orders, err := s.client.FetchOrders(ctx, apiKey, date, resultNumber)
	if err != nil {
		slog.Error("Error fetching orders", "error", err)
		return nil, err
	}

	if len(orders) == 0 {
		return nil, errors.New("no orders found for this date")
	}

	// 2) Procesar cada orden independientemente
	result := &ProcessResult{
		TotalOrders: len(orders),
		Details:     []OrderStatus{},
		Errors:      []string{},
	}

	logger := slog.Default()

	for i := range orders {
		order := &orders[i]

		result.OrdersProcessed++

		// Comparar usando el módulo compare
		compareResult, err := compare.CompareOrderStatus(order, logger)
		if err != nil {
			// Si no se puede comparar (ej: history < 2), loggear y continuar
			result.OrdersSkipped++
			errMsg := fmt.Sprintf("Order %d: %s", order.ID, err.Error())
			result.Errors = append(result.Errors, errMsg)

			logger.Warn("skipping order due to comparison error",
				"order_id", order.ID,
				"error", err,
			)
			continue
		}

		// Crear detalle del resultado
		statusInfo := OrderStatus{
			OrderID:       fmt.Sprintf("%d", compareResult.OrderID),
			PreviousState: compareResult.OldStatus,
			CurrentState:  compareResult.NewStatus,
			Changed:       compareResult.Changed,
		}

		// Si hubo cambio, encolar para webhook
		if compareResult.Changed {
			result.ChangesDetected++
			s.workerPool.Enqueue(*order)
			result.WebhooksSent++
		}

		result.Details = append(result.Details, statusInfo)
	}

	return result, nil
}
