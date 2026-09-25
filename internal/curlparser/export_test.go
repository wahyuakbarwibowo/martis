package curlparser

import (
	"martis/internal/domain"
	"os"
	"path/filepath"
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

func TestParseAndExportMultipartFieldsAndBodyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "payload.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse("curl -F name=martis -F avatar=@" + path + " --data-binary @" + path + " https://api.test/upload")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.FormFields["name"] != "martis" || parsed.FormFiles["avatar"] != path || parsed.BodyFile != path {
		t.Fatalf("unexpected parsed request: %#v", parsed)
	}
	cmd := Export(domain.RequestPayload{Method: "POST", URL: "https://api.test/upload", BodyFile: path})
	if !strings.Contains(cmd, "--data-binary") || !strings.Contains(cmd, "@"+path) {
		t.Fatalf("missing body file export: %s", cmd)
	}
}
