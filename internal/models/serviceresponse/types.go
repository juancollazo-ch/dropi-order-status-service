// internal/models/serviceresponse/types.go
package serviceresponse

// ProcessResult representa el resultado final del procesamiento de órdenes.
type ProcessResult struct {
	TotalOrders     int             `json:"total_orders"`
	OrdersProcessed int             `json:"orders_processed"`
	ChangesDetected int             `json:"changes_detected"`
	WebhooksQueued  int             `json:"webhooks_queued"`
	WebhooksPending int             `json:"webhooks_pending"`
	OrdersSkipped   int             `json:"orders_skipped"`
	Errors          []string        `json:"-"` // No se envía al cliente (solo para logs internos)
	Details         []OrderStatus   `json:"details"`
	FailedWebhooks  []FailedWebhook `json:"failed_webhooks,omitempty"`
	PartialTimeout  bool            `json:"partial_timeout,omitempty"`
}

// FailedWebhook representa los detalles de un webhook que falló después de los reintentos.
type FailedWebhook struct {
	OrderID      string   `json:"order_id"`
	ProductNames []string `json:"product_names"`
	Status       string   `json:"status"`
	WebhookURL   string   `json:"webhook_url"`
	Error        string   `json:"error"`
}

// OrderStatus representa el estado de una orden después de la comparación.
type OrderStatus struct {
	OrderID       string   `json:"order_id"`
	ProductNames  []string `json:"product_names"`
	PreviousState string   `json:"previous_state"`
	CurrentState  string   `json:"current_state"`
	Changed       bool     `json:"changed"`
}
