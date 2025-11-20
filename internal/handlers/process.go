package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/service"
)

type ProcessHandler struct {
	svc *service.OrderService
}

func NewProcessHandler(svc *service.OrderService) *ProcessHandler {
	return &ProcessHandler{svc: svc}
}

// Request esperado desde el cliente
type ProcessRequest struct {
	APIKey string `json:"api_key"`
	Date   string `json:"date"`
}

func (h *ProcessHandler) ProcessOrders(w http.ResponseWriter, r *http.Request) {
	var req ProcessRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.APIKey == "" || req.Date == "" {
		http.Error(w, "api_key and date are required", http.StatusBadRequest)
		return
	}

	// Aquí llamamos al servicio
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	result, err := h.svc.HandleOrderRequest(ctx, req.APIKey, req.Date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
