package notion

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		contains string
	}{
		{
			name:     "with hint",
			err:      newAPIError(401, "unauthorized", "API token is invalid"),
			contains: "check that NOTION_TOKEN",
		},
		{
			name:     "403 permission",
			err:      newAPIError(403, "restricted_resource", "not shared"),
			contains: "share the page with your integration",
		},
		{
			name:     "404 not found",
			err:      newAPIError(404, "object_not_found", "page not found"),
			contains: "verify the page/database ID",
		},
		{
			name:     "429 rate limit",
			err:      newAPIError(429, "rate_limited", "too many requests"),
			contains: "rate limited",
		},
		{
			name:     "500 server error",
			err:      newAPIError(500, "internal_server_error", "internal error"),
			contains: "Notion server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if !containsStr(msg, tt.contains) {
				t.Errorf("expected error to contain %q, got %q", tt.contains, msg)
			}
		})
	}
}

func TestIsAPIError(t *testing.T) {
	apiErr := newAPIError(401, "unauthorized", "bad token")
	wrapped := fmt.Errorf("outer: %w", apiErr)

	got, ok := IsAPIError(wrapped)
	if !ok {
		t.Fatal("expected IsAPIError to return true")
	}
	if got.StatusCode != 401 {
		t.Errorf("expected 401, got %d", got.StatusCode)
	}
}

func TestIsAPIError_NonAPIError(t *testing.T) {
	_, ok := IsAPIError(errors.New("plain error"))
	if ok {
		t.Error("expected IsAPIError to return false for non-API error")
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name   string
		code   int
		check  func(error) bool
		expect bool
	}{
		{"IsRateLimit true", 429, IsRateLimit, true},
		{"IsRateLimit false", 401, IsRateLimit, false},
		{"IsUnauthorized true", 401, IsUnauthorized, true},
		{"IsUnauthorized false", 403, IsUnauthorized, false},
		{"IsForbidden true", 403, IsForbidden, true},
		{"IsForbidden false", 401, IsForbidden, false},
		{"IsNotFound true", 404, IsNotFound, true},
		{"IsNotFound false", 200, IsNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newAPIError(tt.code, "", "test")
			if got := tt.check(err); got != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, got)
			}
		})
	}
}

func TestParseAPIError_ValidJSON(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"code":    "unauthorized",
		"message": "API token is invalid.",
	})
	apiErr := parseAPIError(401, body)
	if apiErr.Code != "unauthorized" {
		t.Errorf("expected code 'unauthorized', got %q", apiErr.Code)
	}
	if apiErr.Message != "API token is invalid." {
		t.Errorf("expected message 'API token is invalid.', got %q", apiErr.Message)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestParseAPIError_InvalidJSON(t *testing.T) {
	apiErr := parseAPIError(500, []byte("not json"))
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "not json" {
		t.Errorf("expected raw body as message, got %q", apiErr.Message)
	}
}

func TestDo_401Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "unauthorized",
			"message": "API token is invalid.",
		})
	}))
	defer srv.Close()

	client := NewClientWithBase(srv.URL, "bad-token")
	_, err := client.GetDatabase("test-id")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
	if !IsUnauthorized(err) {
		t.Error("expected IsUnauthorized to be true")
	}
}

func TestDo_403Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "restricted_resource",
			"message": "Not shared with integration.",
		})
	}))
	defer srv.Close()

	client := NewClientWithBase(srv.URL, "token")
	_, err := client.GetDatabase("test-id")
	if !IsForbidden(err) {
		t.Errorf("expected IsForbidden, got %v", err)
	}
}

func TestDo_429Retry(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(429)
			json.NewEncoder(w).Encode(map[string]string{
				"code":    "rate_limited",
				"message": "Rate limited.",
			})
			return
		}
		// Third attempt succeeds
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "db-123",
		})
	}))
	defer srv.Close()

	client := NewClientWithBase(srv.URL, "token")
	// Override retry delay for test speed — we test the logic, not the timing
	client.retryDelay = 1 * time.Millisecond
	resp, err := client.do("GET", "/databases/test", nil)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if resp["id"] != "db-123" {
		t.Errorf("expected id 'db-123', got %v", resp["id"])
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestDo_429ExhaustedRetries(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "rate_limited",
			"message": "Rate limited.",
		})
	}))
	defer srv.Close()

	client := NewClientWithBase(srv.URL, "token")
	client.retryDelay = 1 * time.Millisecond
	_, err := client.do("GET", "/databases/test", nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	// maxRetries=3, so 1 initial + 3 retries = 4 attempts
	if got := atomic.LoadInt32(&attempts); got != 4 {
		t.Errorf("expected 4 attempts, got %d", got)
	}
	if !IsRateLimit(err) {
		t.Errorf("expected rate limit error in chain, got: %v", err)
	}
}

func TestDo_500Retry(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(502)
			json.NewEncoder(w).Encode(map[string]string{
				"code":    "internal_server_error",
				"message": "Bad gateway",
			})
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "ok"})
	}))
	defer srv.Close()

	client := NewClientWithBase(srv.URL, "token")
	client.retryDelay = 1 * time.Millisecond
	resp, err := client.do("GET", "/test", nil)
	if err != nil {
		t.Fatalf("expected success after server error retry, got: %v", err)
	}
	if resp["id"] != "ok" {
		t.Errorf("expected id 'ok', got %v", resp["id"])
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
