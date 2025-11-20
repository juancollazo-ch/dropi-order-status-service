package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/api"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/service"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/webhook"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/worker"
)

type ProcessRequest struct {
	APIKey string `json:"api_key"`
	Date   string `json:"date"`
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// ---------------------------------------------
// MAIN: inicia servidor, workers y dependencias
// ---------------------------------------------
func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// ---- Inicializar dependencias ----

	dropiClient, err := api.NewDropiClient()
	if err != nil {
		logger.Error("Failed to start Dropi client", "error", err)
		os.Exit(1)
	}

	sender := webhook.NewSender()

	workerPool := worker.NewWorkerPool(sender, 50) // 50 workers para manejar carga concurrente
	workerCtx := context.Background()
	workerPool.Start(workerCtx)

	orderService := service.NewOrderService(dropiClient, workerPool)

	// ---- HTTP Mux ----

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/process", withLogging(processHandler(orderService)))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,  // Aumentado para manejar payloads grandes
		WriteTimeout: 60 * time.Second,  // Aumentado para procesamiento de 50 órdenes
		IdleTimeout:  120 * time.Second, // Aumentado para conexiones persistentes
	}

	logger.Info("Server started", "port", port)

	if err := server.ListenAndServe(); err != nil {
		logger.Error("Server stopped", "error", err.Error())
	}
}

// ---------------------------------------------
// MIDDLEWARE LOGGING
// ---------------------------------------------
func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		logger.Info("Request started",
			"method", r.Method,
			"path", r.URL.String(),
			"ip", r.RemoteAddr,
		)

		next(w, r)

		logger.Info("Request completed",
			"method", r.Method,
			"path", r.URL.String(),
			"ip", r.RemoteAddr,
			"duration", time.Since(start),
		)
	}
}

// ---------------------------------------------
// HEALTH CHECK MEJORADO
// ---------------------------------------------
func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":  "healthy",
		"service": "dropi-order-status-service",
		"version": "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// ---------------------------------------------
// VALIDACIÓN DE FECHA
// ---------------------------------------------
func isValidDateFormat(date string) bool {
	// Formato esperado: YYYY-MM-DD
	if len(date) != 10 {
		return false
	}
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

// ---------------------------------------------
// PROCESS HANDLER (usa OrderService + WorkerPool)
// ---------------------------------------------
func processHandler(orderService *service.OrderService) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()

		var req ProcessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Invalid JSON", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.APIKey == "" || req.Date == "" {
			http.Error(w, "api_key and date are required", http.StatusBadRequest)
			return
		}

		// Validar formato de fecha (YYYY-MM-DD)
		if !isValidDateFormat(req.Date) {
			logger.Error("Invalid date format", "date", req.Date)
			http.Error(w, "date must be in format YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		logger.Info("Processing request", "date", req.Date)

		result, err := orderService.HandleOrderRequest(ctx, req.APIKey, req.Date)
		if err != nil {
			logger.Error("Processing error", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)

		logger.Info("Process completed successfully",
			"orders", result.TotalOrders,
			"changes", result.ChangesDetected,
			"webhooks_queued", result.WebhooksSent,
		)
	}
}
