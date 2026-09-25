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
