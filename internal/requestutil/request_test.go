package requestutil

import (
	"encoding/base64"
	"martis/internal/domain"
	"testing"
)

func TestPrepareAuthenticationAndVariables(t *testing.T) {
	p := domain.RequestPayload{URL: "{{base}}/users", Auth: domain.AuthConfig{Mode: "basic", Username: "alice", Password: "secret"}}
	got, err := Prepare(p, map[string]string{"base": "https://api.test"})
	if err != nil {
		t.Fatal(err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	if got.URL != "https://api.test/users" || got.HeaderAuth != want {
		t.Fatalf("unexpected prepared request: %#v", got)
	}
}

func TestWithQueryReplacesOldQuery(t *testing.T) {
	got, err := WithQuery("https://api.test/users?old=x#frag", []domain.KeyValue{{Key: "page", Value: "2"}, {Key: "q", Value: "two words"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://api.test/users?page=2&q=two+words#frag" {
		t.Fatalf("unexpected URL %q", got)
	}
}

func TestAssertStatusAndJSONPath(t *testing.T) {
	response := domain.ResponseResult{StatusCode: 200, Body: `{"data":{"id":1}}`}
	if failed := Assert(response, "Status == 200\njson.data.id != nil"); len(failed) != 0 {
		t.Fatalf("unexpected failures: %v", failed)
	}
	if failed := Assert(response, "Status == 201\njson.missing != nil"); len(failed) != 2 {
		t.Fatalf("expected two failures, got %v", failed)
	}
}

func TestCaptureJSONValues(t *testing.T) {
	response := domain.ResponseResult{StatusCode: 200, Body: `{"auth":{"token":"abc","ttl":60}}`}
	got, failed := Capture(response, "Status == 200\nset token = json.auth.token\nset ttl = json.auth.ttl\nset gone = json.nope\nset 1bad = json.auth")
	if got["token"] != "abc" || got["ttl"] != "60" || len(got) != 2 || len(failed) != 2 {
		t.Fatalf("unexpected capture %v, failures %v", got, failed)
	}
	if failed := Assert(response, "set token = json.auth.token"); len(failed) != 0 {
		t.Fatalf("set lines must not fail assertions: %v", failed)
	}
}

func TestJSONPathArraysAndObjects(t *testing.T) {
	body := `{"data":[{"id":1,"tags":["a"]}]}`
	if got, ok := JSONPath(body, "json.data.0.tags"); !ok || got != "[\n  \"a\"\n]" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	for _, path := range []string{"json.data.1", "json.data.x", "json.data.0.id.deep"} {
		if _, ok := JSONPath(body, path); ok {
			t.Fatalf("%s should be missing", path)
		}
	}
}
