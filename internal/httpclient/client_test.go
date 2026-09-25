package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"martis/internal/domain"
)

func TestHttpClientDo(t *testing.T) {
	// Mock HTTP Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewClient("TestAgent/1.0")

	payload := domain.RequestPayload{
		Method:      "GET",
		URL:         server.URL,
		HeaderAuth:  "Bearer test-token",
		TimeoutSecs: 2 * time.Second,
	}

	resp := client.Do(payload)
	if resp.Err != nil {
		t.Fatalf("unexpected HTTP client error: %v", resp.Err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if resp.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", resp.Headers.Get("Content-Type"))
	}
}
