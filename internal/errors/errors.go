package errors

import (
	"fmt"
	"net/http"
)

// AppError representa un error de aplicación con código HTTP y contexto
type AppError struct {
	Code       int                    `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Internal   error                  `json:"-"` // No se expone al cliente
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Retryable  bool                   `json:"retryable"`
	StatusCode int                    `json:"-"` // HTTP status code
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

// NewAppError crea un nuevo error de aplicación
func NewAppError(statusCode int, code int, message string, internal error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Internal:   internal,
		StatusCode: statusCode,
		Metadata:   make(map[string]interface{}),
		Retryable:  false,
	}
}

// WithDetails agrega detalles adicionales al error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithMetadata agrega metadata al error
func (e *AppError) WithMetadata(key string, value interface{}) *AppError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithRetryable marca el error como reintentable
func (e *AppError) WithRetryable(retryable bool) *AppError {
	e.Retryable = retryable
	return e
}

// Errores predefinidos para la API externa (Dropi)
var (
	// Errores de cliente (4xx)
	ErrBadRequest = func(details string, err error) *AppError {
		return NewAppError(http.StatusBadRequest, 40000, "Invalid request", err).
			WithDetails(details)
	}

	ErrUnauthorized = func(details string, err error) *AppError {
		return NewAppError(http.StatusUnauthorized, 40100, "Authentication failed", err).
			WithDetails(details).
			WithRetryable(false)
	}

	ErrForbidden = func(details string, err error) *AppError {
		return NewAppError(http.StatusForbidden, 40300, "Access denied", err).
			WithDetails(details).
			WithRetryable(false)
	}

	ErrNotFound = func(details string, err error) *AppError {
		return NewAppError(http.StatusNotFound, 40400, "Resource not found", err).
			WithDetails(details)
	}

	ErrRateLimited = func(details string, err error) *AppError {
		return NewAppError(http.StatusTooManyRequests, 42900, "Rate limit exceeded", err).
			WithDetails(details).
			WithRetryable(true)
	}

	// Errores de servidor (5xx)
	ErrInternalServer = func(details string, err error) *AppError {
		return NewAppError(http.StatusInternalServerError, 50000, "Internal server error", err).
			WithDetails(details).
			WithRetryable(true)
	}

	ErrServiceUnavailable = func(details string, err error) *AppError {
		return NewAppError(http.StatusServiceUnavailable, 50300, "Service temporarily unavailable", err).
			WithDetails(details).
			WithRetryable(true)
	}

	ErrGatewayTimeout = func(details string, err error) *AppError {
		return NewAppError(http.StatusGatewayTimeout, 50400, "Request timeout", err).
			WithDetails(details).
			WithRetryable(true)
	}

	// Errores de negocio
	ErrValidation = func(details string, err error) *AppError {
		return NewAppError(http.StatusUnprocessableEntity, 42200, "Validation error", err).
			WithDetails(details)
	}

	ErrExternalAPI = func(statusCode int, details string, err error) *AppError {
		return NewAppError(http.StatusBadGateway, 50200, "External API error", err).
			WithDetails(details).
			WithMetadata("external_status_code", statusCode).
			WithRetryable(statusCode >= 500)
	}
)

// IsRetryable verifica si un error es reintentable
func IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Retryable
	}
	return false
}

// GetStatusCode obtiene el código HTTP de un error
func GetStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}
