package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "collection.json")
	if err := os.WriteFile(p, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPostmanImport(t *testing.T) {
	p := writeFixture(t, `{"info":{"name":"Demo"},"item":[{"name":"Users","item":[{"name":"List users","request":{"method":"GET","url":{"raw":"https://api.test/users"}}}]}]}`)
	col, err := File(p)
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "Demo" || len(col.Folders) != 1 || len(col.Folders[0].Items) != 1 || col.Folders[0].Items[0].URL != "https://api.test/users" {
		t.Fatalf("unexpected Postman import: %#v", col)
	}
}

func TestOpenAPIImport(t *testing.T) {
	p := writeFixture(t, `{"openapi":"3.0.0","info":{"title":"API"},"servers":[{"url":"https://api.test"}],"paths":{"/users":{"get":{"summary":"List users"},"post":{"summary":"Create user"}}}}`)
	col, err := File(p)
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "API" || len(col.Folders) != 1 || len(col.Folders[0].Items) != 2 {
		t.Fatalf("unexpected OpenAPI import: %#v", col)
	}
}

func TestPostmanEnvironmentImport(t *testing.T) {
	p := writeFixture(t, `{"name":"prod","values":[{"key":"token","value":"a b","enabled":true},{"key":"base","value":"https://api.test"},{"key":"off","value":"x","enabled":false},{"key":"port","value":8080}]}`)
	name, dotenv, ok, err := PostmanEnvironment(p)
	if err != nil || !ok || name != "prod" {
		t.Fatalf("name=%q ok=%v err=%v", name, ok, err)
	}
	if dotenv != "base=\"https://api.test\"\nport=\"8080\"\ntoken=\"a b\"\n" {
		t.Fatalf("unexpected dotenv %q", dotenv)
	}
	if _, _, ok, _ := PostmanEnvironment(writeFixture(t, `{"info":{"name":"Demo"},"item":[]}`)); ok {
		t.Fatal("collection must not be detected as environment")
	}
}

func TestIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		v := id()
		if seen[v] {
			t.Fatalf("duplicate id %s", v)
		}
		seen[v] = true
	}
}
