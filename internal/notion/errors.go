package notion

import (
	"errors"
	"fmt"
)

// APIError represents a structured error response from the Notion API.
type APIError struct {
	StatusCode int
	Code       string // Notion error code (e.g., "unauthorized", "object_not_found")
	Message    string // Notion error message
	Hint       string // User-friendly suggestion
}

func (e *APIError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("Notion API error %d: %s — %s", e.StatusCode, e.Message, e.Hint)
	}
	return fmt.Sprintf("Notion API error %d: %s", e.StatusCode, e.Message)
}

// IsAPIError checks if an error is a Notion API error and returns it.
func IsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// IsRateLimit checks if the error is a 429 rate limit error.
func IsRateLimit(err error) bool {
	apiErr, ok := IsAPIError(err)
	return ok && apiErr.StatusCode == 429
}

// IsUnauthorized checks if the error is a 401 auth error.
func IsUnauthorized(err error) bool {
	apiErr, ok := IsAPIError(err)
	return ok && apiErr.StatusCode == 401
}

// IsForbidden checks if the error is a 403 permission error.
func IsForbidden(err error) bool {
	apiErr, ok := IsAPIError(err)
	return ok && apiErr.StatusCode == 403
}

// IsNotFound checks if the error is a 404 not found error.
func IsNotFound(err error) bool {
	apiErr, ok := IsAPIError(err)
	return ok && apiErr.StatusCode == 404
}

// newAPIError creates an APIError with a user-friendly hint based on status code.
func newAPIError(statusCode int, code, message string) *APIError {
	hint := ""
	switch statusCode {
	case 401:
		hint = "check that NOTION_TOKEN is set and valid"
	case 403:
		hint = "share the page with your integration (page → ··· → Connections → add integration)"
	case 404:
		hint = "verify the page/database ID exists and is shared with your integration"
	case 409:
		hint = "conflict — the resource was modified by another process, try again"
	case 429:
		hint = "rate limited by Notion — retrying automatically"
	case 500, 502, 503:
		hint = "Notion server error — try again in a few seconds"
	}
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Hint:       hint,
	}
}
