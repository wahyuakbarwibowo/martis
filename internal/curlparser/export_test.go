package curlparser

import (
	"martis/internal/domain"
	"strings"
	"testing"
)

func TestExportRoundTripsJSONAndHeaders(t *testing.T) {
	p := domain.RequestPayload{Method: "POST", URL: "https://api.example.test/users", HeaderKey: "Content-Type", HeaderVal: "application/json", Headers: []domain.KeyValue{{Key: "X-Token", Value: "a b"}}, BodyRaw: `{"name":"Martis"}`}
	got, err := Parse(Export(p))
	if err != nil {
		t.Fatal(err)
	}
	if got.Method != "POST" || got.URL != p.URL || got.Headers["Content-Type"] != "application/json" || got.Headers["X-Token"] != "a b" || got.Body != p.BodyRaw {
		t.Fatalf("round trip mismatch: %#v", got)
	}
}

func TestExportMultipartFile(t *testing.T) {
	cmd := Export(domain.RequestPayload{Method: "POST", URL: "https://api.test/upload", BodyType: "form", FormKey: "avatar", FormPath: "/tmp/my file.png"})
	if !strings.Contains(cmd, "-F 'avatar=@/tmp/my file.png'") {
		t.Fatalf("missing multipart field: %s", cmd)
	}
	got, err := Parse(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if got.FormKey != "avatar" || got.FormPath != "/tmp/my file.png" {
		t.Fatalf("unexpected multipart parse: %#v", got)
	}
}

func TestParseMultipartFileAttributes(t *testing.T) {
	got, err := Parse(`curl -F 'avatar=@./photo.png;type=image/png' https://api.test/upload`)
	if err != nil {
		t.Fatal(err)
	}
	if got.FormKey != "avatar" || got.FormPath != "./photo.png" || got.Method != "POST" {
		t.Fatalf("unexpected multipart parse: %#v", got)
	}
}
