package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/api"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/handlers"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/service"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/webhook"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Context Keys (tipado seguro)
type contextKey string

const traceIDKey contextKey = "trace_id"

// Convertir niveles de Zap a severidad de GCP Cloud Logging
func zapLevelToGCPSeverity(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch level {
	case zapcore.DebugLevel:
		enc.AppendString("DEBUG")
	case zapcore.InfoLevel:
		enc.AppendString("INFO")
	case zapcore.WarnLevel:
		enc.AppendString("WARNING")
	case zapcore.ErrorLevel:
		enc.AppendString("ERROR")
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		enc.AppendString("CRITICAL")
	case zapcore.FatalLevel:
		enc.AppendString("EMERGENCY")
	default:
		enc.AppendString("DEFAULT")
	}
}

// MAIN: inicializa servidor, workers y dependencias
func main() {
	// Inicializar Zap Logger con formato compatible con GCP Cloud Logging
	config := zap.NewProductionConfig()

	// Sembrar el generador de números aleatorios para el jitter en los reintentos
	rand.Seed(time.Now().UnixNano()) //

	// Configurar para Cloud Logging (JSON estructurado)
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "severity"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeLevel = zapLevelToGCPSeverity
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
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

	// Inicializar dependencias
	dropiClient, err := api.NewDropiClient()
	if err != nil {
		zap.L().Error("Failed to start Dropi client", zap.Error(err))
		os.Exit(1)
	}

	webhookSender := webhook.NewSender()

	orderService := service.NewOrderService(dropiClient, webhookSender)
	processHandler := handlers.NewProcessHandler(orderService)

	// HTTP ROUTES
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

	// GRACEFUL SHUTDOWN
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

// MIDDLEWARE: Logging con Trace ID compatible con GCP
func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extraer Trace ID de Cloud Run
		traceHeader := r.Header.Get("X-Cloud-Trace-Context")
		var traceID string
		if traceHeader != "" {
			// Formato: TRACE_ID/SPAN_ID;o=TRACE_TRUE
			// Solo necesitamos TRACE_ID
			if slashIdx := strings.IndexByte(traceHeader, '/'); slashIdx != -1 { //
				traceID = traceHeader[:slashIdx]
			} else {
				traceID = traceHeader // No hay slash, asumir que todo el encabezado es el trace ID
			}
		}
		if traceID == "" {
			traceID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
		}

		// Obtener Project ID para el formato completo de trace
		projectID := os.Getenv("GCP_PROJECT")
		if projectID == "" {
			projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}

		// Guardar traceID en contexto
		ctx := context.WithValue(r.Context(), traceIDKey, traceID)

		// Log con formato compatible con Cloud Logging
		logFields := []zap.Field{
			zap.String("httpRequest.requestMethod", r.Method),
			zap.String("httpRequest.requestUrl", r.URL.Path),
			zap.String("httpRequest.remoteIp", r.RemoteAddr),
			zap.String("httpRequest.userAgent", r.UserAgent()),
		}

		// Agregar trace en formato GCP si tenemos projectID
		if projectID != "" && traceID != "" {
			logFields = append(logFields, zap.String("logging.googleapis.com/trace", fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)))
		}

		zap.L().Info("Request started", logFields...)

		next(w, r.WithContext(ctx))

		duration := time.Since(start)

		completedFields := []zap.Field{
			zap.String("httpRequest.requestMethod", r.Method),
			zap.String("httpRequest.requestUrl", r.URL.Path),
			zap.Int64("httpRequest.latency.milliseconds", duration.Milliseconds()),
			zap.Float64("httpRequest.latency.seconds", duration.Seconds()),
		}

		if projectID != "" && traceID != "" {
			completedFields = append(completedFields, zap.String("logging.googleapis.com/trace", fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)))
		}

		zap.L().Info("Request completed", completedFields...)
	}
}

// HEALTH CHECK
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:  "healthy",
		Service: "dropi-order-status-service",
		Version: "1.2.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
