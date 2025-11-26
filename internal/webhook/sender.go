package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
	"github.com/juancollazo-ch/dropi-order-status-service/internal/retry"
	"go.uber.org/zap"
)

type Sender struct {
	httpClient *http.Client
	baseURL    string
}

// NewSender construye un nuevo Webhook Sender leyendo la variable WEBHOOK_BASE_URL.
func NewSender() *Sender {
	base := os.Getenv("WEBHOOK_BASE_URL")
	if base == "" {
		base = "https://default-webhook.com" // fallback seguro
	}

	// Transport optimizado para webhooks
	transport := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Sender{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		baseURL: strings.TrimRight(base, "/"),
	}
}

// BuildWebhookURL asegura que los slashes se manejen correctamente.
func (s *Sender) BuildWebhookURL(suffix string) (string, error) {
	if suffix == "" {
		return "", fmt.Errorf("webhook suffix is required")
	}

	cleanSuffix := strings.Trim(suffix, "/")
	full := s.baseURL + "/" + cleanSuffix

	return full, nil
}

// SendWebhook envía un webhook a un endpoint dinámico.
func (s *Sender) SendWebhook(ctx context.Context, order models.DropiOrder, webhookSuffix string) error {
	url, err := s.BuildWebhookURL(webhookSuffix)
	if err != nil {
		return err
	}

	// Usar el método del modelo para convertir
	payload := order.ToWebhookPayload()

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling webhook payload: %w", err)
	}

	zap.L().Info("sending webhook",
		zap.String("url", url),
		zap.Int64("order_id", order.ID),
		zap.String("status", order.Status),
	)

	attemptCount := 0

	err = retry.WithRetry(ctx, 3, time.Second, func() error {
		attemptCount++

		zap.L().Info("webhook attempt",
			zap.String("url", url),
			zap.Int64("order_id", order.ID),
			zap.Int("attempt", attemptCount),
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("error creating webhook request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Retry-Attempt", fmt.Sprintf("%d", attemptCount))

		resp, err := s.httpClient.Do(req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) { //
				zap.L().Warn("webhook request cancelled due to context",
					zap.String("url", url),
					zap.Int64("order_id", order.ID),
					zap.Int("attempt", attemptCount),
					zap.Error(err),
				)
				return err // Propagar el error del contexto
			}
			zap.L().Warn("webhook request failed",
				zap.String("url", url),
				zap.Int64("order_id", order.ID),
				zap.Int("attempt", attemptCount),
				zap.Error(err),
			)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			zap.L().Info("webhook sent successfully",
				zap.String("url", url),
				zap.Int64("order_id", order.ID),
				zap.Int("status_code", resp.StatusCode),
				zap.Int("attempt", attemptCount),
			)
			return nil
		}

		err = fmt.Errorf("webhook failed with status %d", resp.StatusCode)
		zap.L().Warn("webhook failed",
			zap.String("url", url),
			zap.Int64("order_id", order.ID),
			zap.Int("status_code", resp.StatusCode),
			zap.Int("attempt", attemptCount),
		)
		return err
	})

	if err != nil {
		// Distinguir entre cancelación de contexto y otros errores
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) { //
			zap.L().Warn("webhook sending cancelled after some retries due to context",
				zap.String("url", url),
				zap.Int64("order_id", order.ID),
				zap.Int("total_attempts", attemptCount),
				zap.Error(err),
			)
		} else {
			zap.L().Error("webhook failed after all retries",
				zap.String("url", url),
				zap.Int64("order_id", order.ID),
				zap.Int("total_attempts", attemptCount),
				zap.Error(err),
			)
		}
	}

	return err
}
