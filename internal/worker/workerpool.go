package worker

import (
	"context"
	"log/slog"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/webhook"
)

type WorkerPool struct {
	jobs    chan models.DropiOrder
	workers int
	sender  *webhook.Sender
}

func NewWorkerPool(sender *webhook.Sender, workers int) *WorkerPool {
	if workers <= 0 {
		workers = 5 // default
	}

	return &WorkerPool{
		jobs:    make(chan models.DropiOrder, 10000), // buffer de 10000 para 100 clientes × 50 órdenes
		workers: workers,
		sender:  sender,
	}
}

// Iniciar N workers
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workers; i++ {
		go wp.worker(ctx, i)
	}
}

// Encolar un trabajo
func (wp *WorkerPool) Enqueue(order models.DropiOrder) {
	wp.jobs <- order
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
	slog.Info("worker iniciado", "id", id)

	for {
		select {
		case <-ctx.Done():
			slog.Warn("worker apagado", "id", id)
			return

		case order := <-wp.jobs:

			if wp.sender == nil {
				slog.Error("webhook sender nil")
				continue
			}

			if err := wp.sender.Send(order); err != nil {
				slog.Error("error enviando webhook", "order_id", order.ID, "error", err)
			} else {
				slog.Info("webhook enviado", "order_id", order.ID)
			}
		}
	}
}
