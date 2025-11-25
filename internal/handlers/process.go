package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/service"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/validator"
	"go.uber.org/zap"
)

type ProcessHandler struct {
	svc       *service.OrderService
	validator *validator.RequestValidator
}

func NewProcessHandler(svc *service.OrderService) *ProcessHandler {
	return &ProcessHandler{
		svc:       svc,
		validator: validator.NewRequestValidator(),
	}
}

func (h *ProcessHandler) ProcessOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Timeout de 180s (3 minutos) para manejar el peor caso:
	// 150 órdenes con cambios = 150 webhooks con posibles reintentos
	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	var req models.ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		zap.L().Error("Invalid JSON", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.APIKey == "" || req.Date == "" {
		http.Error(w, "api_key and date are required", http.StatusBadRequest)
		return
	}

	// Validar formato de fecha (YYYY-MM-DD)
	if !validator.IsValidDate(req.Date) {
		zap.L().Error("Invalid date format", zap.String("date", req.Date))
		http.Error(w, "date must be in format YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Validar parámetros dinámicos
	if err := h.validator.ValidateRequest(&req); err != nil {
		zap.L().Error("Request validation failed",
			zap.Error(err),
			zap.String("dropi_country_suffix", req.DropiCountrySuffix),
			zap.String("webhook_suffix", req.WebhookSuffix),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	zap.L().Info("Processing request",
		zap.String("date", req.Date),
		zap.String("dropi_country_suffix", req.DropiCountrySuffix),
		zap.String("webhook_suffix", req.WebhookSuffix),
	)

	result, err := h.svc.HandleOrderRequest(
		ctx,
		req.APIKey,
		req.Date,
		req.DropiCountrySuffix,
		req.WebhookSuffix,
		req.DateUtil,
	)

	if err != nil {
		zap.L().Error("Processing error", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	zap.L().Info("Process completed successfully",
		zap.Int("orders", result.TotalOrders),
		zap.Int("changes", result.ChangesDetected),
		zap.Int("webhooks_queued", result.WebhooksQueued),
		zap.Bool("partial_timeout", result.PartialTimeout),
	)
}
