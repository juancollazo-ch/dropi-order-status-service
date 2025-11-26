package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// RequestValidator validates request parameters for security and format compliance
type RequestValidator struct {
	countrySuffixRegex *regexp.Regexp
	validCountries     map[string]bool
}

// NewRequestValidator creates a new RequestValidator instance
func NewRequestValidator() *RequestValidator {
	return &RequestValidator{
		// Accepts: "co", "mx", "cl" (2 letters) or "py.com" (2 letters + dot + 2-3 letters)
		countrySuffixRegex: regexp.MustCompile(`^(?:[a-z]{2}|com\.py)$`),
		// Lista de países soportados por Dropi
		validCountries: map[string]bool{
			"co":     true, // Colombia
			"mx":     true, // México
			"cl":     true, // Chile
			"ar":     true, // Argentina
			"ec":     true, // Ecuador
			"gt":     true, // Guatemala
			"pa":     true, // Panamá
			"com.py": true, // Paraguay (caso especial)
			"pe":     true, // Perú
		},
	}
}

// ValidateCountrySuffix validates the country suffix for Dropi API
func (v *RequestValidator) ValidateCountrySuffix(suffix string) error {
	if suffix == "" {
		return errors.New("dropi_country_suffix is required")
	}

	// Validar formato
	if !v.countrySuffixRegex.MatchString(suffix) {
		return errors.New("dropi_country_suffix must be 2 lowercase letters or special format (e.g., 'co', 'mx', 'py.com')")
	}

	// Validar que sea un país soportado
	if !v.validCountries[suffix] {
		return fmt.Errorf("dropi_country_suffix '%s' is not supported. Valid countries: co, mx, cl, ar, ec, gt, pa, py.com, pe", suffix)
	}

	return nil
}

// ValidateWebhookSuffix validates the webhook suffix for security
func (v *RequestValidator) ValidateWebhookSuffix(suffix string) error {
	if suffix == "" {
		return errors.New("webhook_suffix is required")
	}

	// Prevent path traversal attacks
	if strings.Contains(suffix, "..") {
		return errors.New("webhook_suffix cannot contain '..'")
	}

	// Prevent dangerous characters that could be used for injection attacks
	dangerousChars := []string{"<", ">", "\"", "'", ";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", "\\", "\n", "\r", "\t"}
	for _, char := range dangerousChars {
		if strings.Contains(suffix, char) {
			return errors.New("webhook_suffix contains invalid characters")
		}
	}

	return nil
}

// ProcessRequest represents the request structure for validation
type ProcessRequest interface {
	GetDropiCountrySuffix() string
	GetWebhookSuffix() string
}

// ValidateRequest validates the entire request
func (v *RequestValidator) ValidateRequest(req ProcessRequest) error {
	if err := v.ValidateCountrySuffix(req.GetDropiCountrySuffix()); err != nil {
		return err
	}
	if err := v.ValidateWebhookSuffix(req.GetWebhookSuffix()); err != nil {
		return err
	}
	return nil
}
