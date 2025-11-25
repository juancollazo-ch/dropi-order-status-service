package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/juancollazo-ch/dropi-order-status-service/internal/models"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

type DropiClient struct {
	http           *http.Client
	baseURL        string
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewDropiClient lee la URL base desde variable de entorno
func NewDropiClient() (*DropiClient, error) {
	baseURL := os.Getenv("DROPI_API_BASE_URL")
	if baseURL == "" {
		legacyURL := os.Getenv("DROPI_BASE_URL")
		if legacyURL != "" {
			zap.L().Warn("Using legacy DROPI_BASE_URL, please migrate to DROPI_API_BASE_URL")
			baseURL = extractBaseURL(legacyURL)
		} else {
			return nil, errors.New("DROPI_API_BASE_URL environment variable is required")
		}
	}

	// Limpiar trailing slash
	baseURL = strings.TrimRight(baseURL, "/")

	// Transport optimizado
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	// Cliente HTTP
	client := &http.Client{
		Timeout:   12 * time.Second,
		Transport: transport,
	}

	// Circuit Breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "dropi-api",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
	})

	return &DropiClient{
		http:           client,
		baseURL:        baseURL,
		circuitBreaker: cb,
	}, nil
}

// extractBaseURL extrae "https://api.dropi" de "https://api.dropi.co/integrations/..."
func extractBaseURL(fullURL string) string {
	if idx := strings.Index(fullURL, "api.dropi"); idx != -1 {
		start := fullURL[:idx+9] // "https://api.dropi"
		if dotIdx := strings.Index(fullURL[idx+9:], "/"); dotIdx != -1 {
			return start
		}
		return start
	}
	return fullURL
}

// BuildDropiURL construye la URL completa: {base}.{suffix}/integrations/orders/myorders
func (c *DropiClient) BuildDropiURL(countrySuffix string) (string, error) {
	if countrySuffix == "" {
		return "", errors.New("dropi_country_suffix is required")
	}

	if len(countrySuffix) < 2 {
		return "", fmt.Errorf("invalid country suffix '%s'", countrySuffix)
	}

	return fmt.Sprintf(
		"%s.%s/integrations/orders/myorders",
		c.baseURL,
		countrySuffix,
	), nil
}

func (c *DropiClient) doFetchOrders(
	ctx context.Context,
	apiKey string,
	date string,
	limit int,
	countrySuffix string,
	dateUtil string,
) ([]models.DropiOrder, error) {
	// Verificar si el context ya expiró
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled before request: %w", ctx.Err())
	default:
	}

	start := time.Now()

	if apiKey == "" {
		return nil, errors.New("api_key required")
	}
	if date == "" {
		return nil, errors.New("date is required")
	}
	if countrySuffix == "" {
		return nil, errors.New("dropi_country_suffix is required")
	}
	if limit <= 0 {
		limit = 1
	}

	// Construir URL dinámica
	url, err := c.BuildDropiURL(countrySuffix)
	if err != nil {
		return nil, fmt.Errorf("error building Dropi URL: %w", err)
	}

	// Crear request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	// Query params
	// Query params base
	queryParams := fmt.Sprintf(
		"from=%s&result_number=%d&filter_date_by=%s",
		date, limit, "FECHA%20DE%20CAMBIO%20DE%20ESTATUS",
	)

	// Agregar date_util si está presente
	if dateUtil != "" {
		queryParams += fmt.Sprintf("&until=%s", dateUtil)
	}

	req.URL.RawQuery = queryParams

	zap.L().Info("calling dropi",
		zap.String("url", req.URL.String()),
		zap.String("country_suffix", countrySuffix),
		zap.String("date", date),
		zap.Int("limit", limit),
	)

	// Headers
	req.Header.Set("dropi-integration-key", apiKey)
	req.Header.Set("User-Agent", "PostmanRuntime/7.26.8")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	// Manejo específico de códigos de error
	switch resp.StatusCode {
	case 429:
		// Rate limiting - error temporal
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			zap.L().Warn("rate limited by Dropi",
				zap.String("retry_after", retryAfter),
				zap.String("country_suffix", countrySuffix),
			)
			return nil, fmt.Errorf("rate limited: retry after %s seconds", retryAfter)
		}
		return nil, fmt.Errorf("rate limited by Dropi API")

	case 401, 403:
		// Error de autenticación - error permanente
		zap.L().Error("authentication failed",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, fmt.Errorf("authentication error: invalid API key (status %d)", resp.StatusCode)

	case 503:
		// Service unavailable - error temporal
		zap.L().Warn("dropi service unavailable",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, fmt.Errorf("dropi service temporarily unavailable")

	case 200, 201, 204:
		// Success - continuar
	default:
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("dropi error: status %d", resp.StatusCode)
		}
	}

	var apiResponse models.DropiAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("invalid JSON from Dropi: %w", err)
	}

	zap.L().Info("dropi response received",
		zap.Int("status_code", resp.StatusCode),
		zap.Int("orders_count", len(apiResponse.Objects)),
		zap.Int("total_count", apiResponse.Count),
		zap.Duration("duration", time.Since(start)),
	)

	return apiResponse.Objects, nil
}

func (c *DropiClient) FetchOrders(
	ctx context.Context,
	apiKey string,
	date string,
	limit int,
	countrySuffix string,
	dateUtil string,
) ([]models.DropiOrder, error) {
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.doFetchOrders(ctx, apiKey, date, limit, countrySuffix, dateUtil)
	})

	if err != nil {
		return nil, err
	}

	return result.([]models.DropiOrder), nil
}

// FetchAllOrders obtiene todas las órdenes usando paginación automática
// Útil cuando hay más de 50 órdenes en una fecha
func (c *DropiClient) FetchAllOrders(
	ctx context.Context,
	apiKey string,
	date string,
	countrySuffix string,
	dateUtil string,
) ([]models.DropiOrder, error) {
	const pageSize = 50
	var allOrders []models.DropiOrder
	page := 1
	maxPages := 10 // Límite de seguridad para evitar loops infinitos

	for page <= maxPages {
		// Verificar timeout
		select {
		case <-ctx.Done():
			zap.L().Warn("pagination stopped due to timeout",
				zap.Int("pages_fetched", page-1),
				zap.Int("total_orders", len(allOrders)),
			)
			return allOrders, nil // Retornar lo que tenemos hasta ahora
		default:
		}

		orders, err := c.FetchOrders(ctx, apiKey, date, pageSize, countrySuffix, dateUtil)
		if err != nil {
			// Si es la primera página, retornar error
			if page == 1 {
				return nil, err
			}
			// Si es una página posterior, retornar lo que tenemos
			zap.L().Warn("pagination stopped due to error",
				zap.Int("page", page),
				zap.Error(err),
			)
			return allOrders, nil
		}

		allOrders = append(allOrders, orders...)

		// Si recibimos menos de pageSize, no hay más páginas
		if len(orders) < pageSize {
			zap.L().Info("pagination completed",
				zap.Int("total_pages", page),
				zap.Int("total_orders", len(allOrders)),
			)
			break
		}

		page++
	}

	if page > maxPages {
		zap.L().Warn("pagination limit reached",
			zap.Int("max_pages", maxPages),
			zap.Int("total_orders", len(allOrders)),
		)
	}

	return allOrders, nil
}
