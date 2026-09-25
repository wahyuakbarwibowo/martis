package environment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndExpand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local.env")
	if err := os.WriteFile(path, []byte("# env\nbase_url=https://api.example.test\ntoken='secret value'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	vars, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Expand("{{base_url}}/users?token={{token}}", vars)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://api.example.test/users?token=secret value" {
		t.Fatalf("unexpected expansion: %s", got)
	}
	if _, err = Expand("{{missing}}", vars); err == nil {
		t.Fatal("expected missing variable error")
	}
	if strings.Contains(got, "{{") {
		t.Fatalf("unexpanded variable: %s", got)
	}
}
