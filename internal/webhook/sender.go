package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
)

type Sender struct {
	http       *http.Client
	webhookURL string
}

// WebhookPayload estructura simplificada para el webhook
type WebhookPayload struct {
	ID                  int64         `json:"id"`
	Status              string        `json:"status"`
	SupplierID          int64         `json:"supplier_id"`
	Dir                 string        `json:"dir"`
	Phone               string        `json:"phone"`
	Email               *string       `json:"email"`
	CreatedAt           string        `json:"created_at"`
	Type                string        `json:"type"`
	TotalOrder          string        `json:"total_order"`
	Notes               *string       `json:"notes"`
	Name                string        `json:"name"`
	Surname             string        `json:"surname"`
	Country             string        `json:"country"`
	State               string        `json:"state"`
	City                string        `json:"city"`
	ZipCode             *string       `json:"zip_code"`
	RateType            string        `json:"rate_type"`
	ShippingCompany     string        `json:"shipping_company"`
	ShippingGuide       string        `json:"shipping_guide"`
	Sticker             string        `json:"sticker"`
	SellerID            *int64        `json:"seller_id"`
	ShopOrderID         string        `json:"shop_order_id"`
	ShopID              int64         `json:"shop_id"`
	ShopOrderNumber     int           `json:"shop_order_number"`
	WarehouseID         int64         `json:"warehouse_id"`
	DNIType             *string       `json:"dni_type"`
	DNI                 *string       `json:"dni"`
	Colonia             *string       `json:"colonia"`
	ExternalID          *string       `json:"external_id"`
	Shop                ShopInfo      `json:"shop"`
	NovedadServientrega *string       `json:"novedad_servientrega"`
	OrderDetails        []string      `json:"orderdetails"` // Array vacío
	Warehouse           WarehouseInfo `json:"warehouse"`
}

type ShopInfo struct {
	ID                    int64   `json:"id"`
	UserID                int64   `json:"user_id"`
	Name                  string  `json:"name"`
	Email                 *string `json:"email"`
	Phone                 *string `json:"phone"`
	Type                  string  `json:"type"`
	CreatedAt             string  `json:"created_at"`
	UpdatedAt             string  `json:"updated_at"`
	DeletedAt             *string `json:"deleted_at"`
	ShopPassword          *string `json:"shop_password"`
	ChangeStatusPendiente bool    `json:"change_status_pendiente"`
	StatusPendiente       *string `json:"status_pendiente"`
	SyncShippingGuide     bool    `json:"sync_shipping_guide"`
	TypeID                int     `json:"type_id"`
	Webhook               *string `json:"webhook"`
}

type WarehouseInfo struct {
	ID   *int64  `json:"id"`
	Name *string `json:"name"`
}

func NewSender() *Sender {
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "https://default-webhook.com/webhook"
	}

	return &Sender{
		http:       &http.Client{Timeout: 10 * time.Second},
		webhookURL: webhookURL,
	}
}

// buildSimplifiedPayload crea el payload simplificado desde DropiOrder
func buildSimplifiedPayload(order models.DropiOrder) WebhookPayload {
	// Convertir SellerID de interface{} a *int64
	var sellerID *int64
	if order.SellerID != nil {
		if val, ok := order.SellerID.(float64); ok {
			id := int64(val)
			sellerID = &id
		}
	}

	// Convertir Novedad de interface{} a *string
	var novedad *string
	if order.Novedad != nil {
		if val, ok := order.Novedad.(string); ok {
			novedad = &val
		}
	}

	// Convertir ShopPassword de interface{} a *string
	var shopPassword *string
	if order.Shop.ShopPassword != nil {
		if val, ok := order.Shop.ShopPassword.(string); ok {
			shopPassword = &val
		}
	}

	// Convertir StatusPendiente de interface{} a *string
	var statusPendiente *string
	if order.Shop.StatusPendiente != nil {
		if val, ok := order.Shop.StatusPendiente.(string); ok {
			statusPendiente = &val
		}
	}

	// Convertir Webhook de interface{} a *string
	var webhook *string
	if order.Shop.Webhook != nil {
		if val, ok := order.Shop.Webhook.(string); ok {
			webhook = &val
		}
	}

	// Convertir Warehouse ID y Name de interface{} a valores concretos
	var warehouseID *int64
	var warehouseName *string

	if order.Warehouse.ID != nil {
		if val, ok := order.Warehouse.ID.(float64); ok {
			id := int64(val)
			warehouseID = &id
		}
	}

	if order.Warehouse.Name != nil {
		if val, ok := order.Warehouse.Name.(string); ok {
			warehouseName = &val
		}
	}

	payload := WebhookPayload{
		ID:                  order.ID,
		Status:              order.Status,
		SupplierID:          order.SupplierID,
		Dir:                 order.Dir,
		Phone:               order.Phone,
		Email:               order.Email,
		CreatedAt:           order.CreatedAt,
		Type:                order.Type,
		TotalOrder:          order.TotalOrder,
		Notes:               order.Notes,
		Name:                order.Name,
		Surname:             order.Surname,
		Country:             order.Country,
		State:               order.State,
		City:                order.City,
		ZipCode:             order.ZipCode,
		RateType:            order.RateType,
		ShippingCompany:     order.ShippingCompany,
		ShippingGuide:       order.ShippingGuide,
		Sticker:             order.Sticker,
		SellerID:            sellerID,
		ShopOrderID:         order.ShopOrderID,
		ShopID:              order.ShopID,
		ShopOrderNumber:     int(order.ShopOrderNumber),
		WarehouseID:         order.WarehouseID,
		DNIType:             order.DNIType,
		DNI:                 order.DNI,
		Colonia:             order.Colonia,
		ExternalID:          order.ExternalID,
		NovedadServientrega: novedad,
		OrderDetails:        []string{}, // Array vacío
		Shop: ShopInfo{
			ID:                    order.Shop.ID,
			UserID:                order.Shop.UserID,
			Name:                  order.Shop.Name,
			Email:                 order.Shop.Email,
			Phone:                 order.Shop.Phone,
			Type:                  order.Shop.Type,
			CreatedAt:             order.Shop.CreatedAt,
			UpdatedAt:             order.Shop.UpdatedAt,
			DeletedAt:             order.Shop.DeletedAt,
			ShopPassword:          shopPassword,
			ChangeStatusPendiente: order.Shop.ChangePendiente,
			StatusPendiente:       statusPendiente,
			SyncShippingGuide:     order.Shop.SyncGuide,
			TypeID:                int(order.Shop.TypeID),
			Webhook:               webhook,
		},
		Warehouse: WarehouseInfo{
			ID:   warehouseID,
			Name: warehouseName,
		},
	}

	return payload
}

func (s *Sender) Send(order models.DropiOrder) error {
	// Crear payload simplificado
	simplifiedPayload := buildSimplifiedPayload(order)

	payload, err := json.Marshal(simplifiedPayload)
	if err != nil {
		return fmt.Errorf("error marshaling order: %w", err)
	}

	// Retry logic: 3 intentos con backoff exponencial
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodPost, s.webhookURL, bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Retry-Attempt", fmt.Sprintf("%d", attempt))

		resp, err := s.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error sending webhook (attempt %d/%d): %w", attempt, maxRetries, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second) // backoff: 1s, 2s, 3s
				continue
			}
			return lastErr
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil // Éxito
		}

		lastErr = fmt.Errorf("webhook failed with status: %d (attempt %d/%d)", resp.StatusCode, attempt, maxRetries)
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return lastErr
}
