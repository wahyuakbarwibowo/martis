package curlparser

import (
	"testing"
)

func TestParse_SimpleGET(t *testing.T) {
	cmd := `curl https://api.example.com/users`
	got, err := Parse(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Method != "GET" {
		t.Errorf("expected GET, got %s", got.Method)
	}
	if got.URL != "https://api.example.com/users" {
		t.Errorf("unexpected URL: %s", got.URL)
	}
}

func TestParse_POSTWithBody(t *testing.T) {
	cmd := `curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"Martis"}'`
	got, err := Parse(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Method != "POST" {
		t.Errorf("expected POST, got %s", got.Method)
	}
	if got.Body != `{"name":"Martis"}` {
		t.Errorf("unexpected body: %s", got.Body)
	}
	if got.Headers["Content-Type"] != "application/json" {
		t.Errorf("unexpected Content-Type: %s", got.Headers["Content-Type"])
	}
}

func TestParse_WithAuth(t *testing.T) {
	cmd := `curl https://api.example.com -H "Authorization: Bearer token123"`
	got, err := Parse(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AuthHeader != "Bearer token123" {
		t.Errorf("unexpected auth: %s", got.AuthHeader)
	}
}

func TestParse_MultilineWithBackslash(t *testing.T) {
	cmd := "curl -X PUT \\\n  https://api.example.com/users/1 \\\n  -H 'Content-Type: application/json' \\\n  -d '{\"status\":\"ok\"}'"
	got, err := Parse(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Method != "PUT" {
		t.Errorf("expected PUT, got %s", got.Method)
	}
}

func TestParse_BodyImpliesPost(t *testing.T) {
	cmd := `curl https://api.example.com -d '{"key":"val"}'`
	got, err := Parse(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Method != "POST" {
		t.Errorf("expected method to be POST when body present, got %s", got.Method)
	}
}

func TestParse_InvalidNoCurl(t *testing.T) {
	_, err := Parse("wget https://example.com")
	if err == nil {
		t.Error("expected error for non-curl command")
	}
}

func TestParse_MissingURL(t *testing.T) {
	_, err := Parse("curl -X GET")
	if err == nil {
		t.Error("expected error when URL is missing")
	}
}
