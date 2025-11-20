package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
)

type DropiClient struct {
	http *http.Client
	base string
}

func NewDropiClient() (*DropiClient, error) {
	base := os.Getenv("DROPI_BASE_URL")
	if base == "" {
		return nil, errors.New("DROPI_BASE_URL is required")
	}

	return &DropiClient{
		http: &http.Client{Timeout: 30 * time.Second}, // Aumentado para manejar respuestas grandes
		base: base,
	}, nil
}

func (c *DropiClient) FetchOrders(
	ctx context.Context,
	apiKey string,
	date string,
	limit int,
) ([]models.DropiOrder, error) {

	if apiKey == "" {
		return nil, errors.New("api_key required")
	}
	if date == "" {
		return nil, errors.New("date is required")
	}
	if limit <= 0 {
		limit = 1
	}

	// Build URL - Construir exactamente como Postman (con %20 para espacios)
	// Postman URL-encode los espacios como %20
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	// Establecer RawQuery con espacios URL-encoded como %20 (igual que Postman)
	req.URL.RawQuery = fmt.Sprintf("from=%s&result_number=%d&filter_date_by=%s",
		date, limit, "FECHA%20DE%20CAMBIO%20DE%20ESTATUS")

	slog.Info("DEBUG Dropi API", "url", req.URL.String(), "rawQuery", req.URL.RawQuery)

	// Headers para Dropi
	req.Header.Set("dropi-integration-key", apiKey)
	req.Header.Set("User-Agent", "PostmanRuntime/7.26.8") // Simular Postman

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dropi error: status %d", resp.StatusCode)
	}

	// Leer el body completo para debugging
	var rawResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("invalid JSON from Dropi: %w", err)
	}

	slog.Info("DEBUG Dropi Response", "keys", getKeys(rawResponse))

	// Dropi puede devolver un objeto con "objects" o directamente un array
	// Intentar extraer el array de órdenes
	var orders []models.DropiOrder

	// Si hay un campo "objects", usar ese
	if objectsField, ok := rawResponse["objects"]; ok {
		objectsJSON, _ := json.Marshal(objectsField)
		if err := json.Unmarshal(objectsJSON, &orders); err != nil {
			return nil, fmt.Errorf("error parsing objects field: %w", err)
		}
	} else {
		// Si no, intentar parsear como array directamente
		responseJSON, _ := json.Marshal(rawResponse)
		if err := json.Unmarshal(responseJSON, &orders); err != nil {
			return nil, fmt.Errorf("error parsing response as array: %w", err)
		}
	}

	return orders, nil
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
