package worker

import (
	"context"
	"log/slog"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/webhook"
)

type WorkerTask struct {
	Order         models.DropiOrder
	WebhookSuffix string
}

type WorkerPool struct {
	jobs    chan WorkerTask
	workers int
	sender  *webhook.Sender
}

func NewWorkerPool(sender *webhook.Sender, workers int) *WorkerPool {
	if workers <= 0 {
		workers = 5
	}

	return &WorkerPool{
		jobs:    make(chan WorkerTask, 10000),
		workers: workers,
		sender:  sender,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workers; i++ {
		go wp.worker(ctx, i)
	}
}

func (wp *WorkerPool) Enqueue(order models.DropiOrder, suffix string) {
	wp.jobs <- WorkerTask{Order: order, WebhookSuffix: suffix}
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
	slog.Info("worker iniciado", "id", id)

	for {
		select {
		case <-ctx.Done():
			slog.Warn("worker apagado", "id", id)
			return

		case task := <-wp.jobs:
			if wp.sender == nil {
				slog.Error("webhook sender nil")
				continue
			}

			if err := wp.sender.SendWebhook(task.Order, task.WebhookSuffix); err != nil {
				slog.Error("error enviando webhook", "order_id", task.Order.ID, "suffix", task.WebhookSuffix, "error", err)
			} else {
				slog.Info("webhook enviado", "order_id", task.Order.ID, "suffix", task.WebhookSuffix)
			}
		}
	}
}
