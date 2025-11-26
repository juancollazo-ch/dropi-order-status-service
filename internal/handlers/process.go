package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	apperrors "github.com/juancollazo-ch/dropi-order-status-service/internal/errors"
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
		// Crear error estructurado para validación
		validationErr := apperrors.ErrValidation(err.Error(), err).
			WithMetadata("dropi_country_suffix", req.DropiCountrySuffix).
			WithMetadata("webhook_suffix", req.WebhookSuffix)

		zap.L().Warn("Request validation failed",
			zap.Error(err),
			zap.String("dropi_country_suffix", req.DropiCountrySuffix),
			zap.String("webhook_suffix", req.WebhookSuffix),
		)

		h.handleError(w, validationErr)
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
		h.handleError(w, err)
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

// handleError maneja errores de forma estructurada y responde con el código HTTP correcto
func (h *ProcessHandler) handleError(w http.ResponseWriter, err error) {
	// Intentar convertir a AppError
	if appErr, ok := err.(*apperrors.AppError); ok {
		// Log estructurado con metadata
		logFields := []zap.Field{
			zap.Int("status_code", appErr.StatusCode),
			zap.Int("error_code", appErr.Code),
			zap.String("message", appErr.Message),
			zap.Bool("retryable", appErr.Retryable),
		}

		// Agregar metadata si existe
		for key, value := range appErr.Metadata {
			logFields = append(logFields, zap.Any(key, value))
		}

		// Log con severidad apropiada
		if appErr.StatusCode >= 500 {
			zap.L().Error("Server error", logFields...)
		} else if appErr.StatusCode >= 400 {
			zap.L().Warn("Client error", logFields...)
		}

		// Responder al cliente con el código correcto
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":      appErr.Code,
				"message":   appErr.Message,
				"details":   appErr.Details,
				"retryable": appErr.Retryable,
				"metadata":  appErr.Metadata,
			},
		})
		return
	}

	// Error genérico (no estructurado)
	zap.L().Error("Unhandled error",
		zap.Error(err),
		zap.String("error_type", "unstructured"),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":      50000,
			"message":   "Internal server error",
			"details":   err.Error(),
			"retryable": true,
		},
	})
}
