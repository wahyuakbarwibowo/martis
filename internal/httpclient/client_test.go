package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestMultipartFileUpload(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(filePath, []byte("hello file"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("attachment")
		if err != nil {
			t.Errorf("form file: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("read upload: %v", err)
		}
		if header.Filename != "upload.txt" || string(data) != "hello file" {
			t.Errorf("unexpected uploaded file %q: %q", header.Filename, data)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	got := NewClient().Do(domain.RequestPayload{Method: "POST", URL: server.URL, BodyType: "form", FormKey: "attachment", FormPath: filePath})
	if got.Err != nil {
		t.Fatalf("upload failed: %v", got.Err)
	}
	if got.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", got.StatusCode)
	}
}

func TestOAuthClientCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "client" || password != "secret" {
			t.Errorf("unexpected client authentication")
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("scope") != "read" {
			t.Errorf("unexpected token form: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","token_type":"Bearer"}`))
	}))
	defer server.Close()
	token, err := OAuthToken(domain.AuthConfig{TokenURL: server.URL, Username: "client", Password: "secret", Scope: "read"})
	if err != nil {
		t.Fatal(err)
	}
	if token != "abc" {
		t.Fatalf("token = %q", token)
	}
}
