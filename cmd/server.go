package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/api"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/handlers"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/service"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/webhook"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/worker"
	"go.uber.org/zap"
)

//
// -------------------------------------------------------
// Context Keys (tipado seguro)
// -------------------------------------------------------
type contextKey string

const traceIDKey contextKey = "trace_id"

//
// -------------------------------------------------------
// MAIN: inicializa servidor, workers y dependencias
// -------------------------------------------------------
func main() {
	// Inicializar Zap Logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Reemplazar logger global
	zap.ReplaceGlobals(logger)

	// Puerto para Cloud Run / local
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	//
	// -----------------------
	// Inicializar dependencias
	// -----------------------
	dropiClient, err := api.NewDropiClient()
	if err != nil {
		zap.L().Error("Failed to start Dropi client", zap.Error(err))
		os.Exit(1)
	}

	sender := webhook.NewSender()

	workerPool := worker.NewWorkerPool(sender, 50) // 50 workers concurrentes
	workerCtx := context.Background()
	workerPool.Start(workerCtx)

	orderService := service.NewOrderService(dropiClient, workerPool)
	processHandler := handlers.NewProcessHandler(orderService)

	//
	// -----------------------
	// HTTP ROUTES
	// -----------------------
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/process", withLogging(processHandler.ProcessOrders))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	//
	// -----------------------
	// GRACEFUL SHUTDOWN
	// -----------------------
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		<-sigChan

		zap.L().Info("Shutting down gracefully...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			zap.L().Error("Graceful shutdown failed", zap.Error(err))
		}

		zap.L().Info("Server exited")
		os.Exit(0)
	}()

	zap.L().Info("Server started", zap.String("port", port))

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zap.L().Error("Server stopped unexpectedly", zap.Error(err))
	}
}

//
// -------------------------------------------------------
// MIDDLEWARE: Logging con Trace ID compatible con GCP
// -------------------------------------------------------
func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Trace ID compatible con Cloud Run / Cloud Logging
		traceID := r.Header.Get("X-Cloud-Trace-Context")
		if traceID == "" {
			traceID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
		}

		// Guardar traceID en contexto
		ctx := context.WithValue(r.Context(), traceIDKey, traceID)

		zap.L().Info("Request started",
			zap.String("trace_id", traceID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("ip", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)

		next(w, r.WithContext(ctx))

		duration := time.Since(start)

		zap.L().Info("Request completed",
			zap.String("trace_id", traceID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.Float64("duration_seconds", duration.Seconds()),
		)
	}
}

//
// -------------------------------------------------------
// HEALTH CHECK
// -------------------------------------------------------
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:  "healthy",
		Service: "dropi-order-status-service",
		Version: "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

