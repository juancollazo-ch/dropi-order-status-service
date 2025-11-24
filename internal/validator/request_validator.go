package validator

import (
	"errors"
	"regexp"
	"strings"
)

// RequestValidator validates request parameters for security and format compliance
type RequestValidator struct {
	countrySuffixRegex *regexp.Regexp
}

// NewRequestValidator creates a new RequestValidator instance
func NewRequestValidator() *RequestValidator {
	return &RequestValidator{
		// Accepts: "co", "mx", "cl" (2 letters) or "py.com" (2 letters + dot + 2-3 letters)
		countrySuffixRegex: regexp.MustCompile(`^[a-z]{2}(\.[a-z]{2,3})?$`),
	}
}

// ValidateCountrySuffix validates the country suffix for Dropi API
func (v *RequestValidator) ValidateCountrySuffix(suffix string) error {
	if suffix == "" {
		return errors.New("dropi_country_suffix is required")
	}
	if !v.countrySuffixRegex.MatchString(suffix) {
		return errors.New("dropi_country_suffix must be 2 lowercase letters or special format (e.g., 'co', 'mx', 'py.com')")
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
