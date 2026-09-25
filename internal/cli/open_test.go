package cli

import (
	"testing"

	"martis/internal/curlparser"
)

func TestInitialCurl(t *testing.T) {
	if InitialCurl(nil) != "" || InitialCurl([]string{"collections"}) != "" {
		t.Fatal("non-request args must not open a request")
	}
	parsed, err := curlparser.Parse(InitialCurl([]string{"curl", "-H", "X-Name: it's me", "-d", `{"a":1}`, "https://api.test/x"}))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.URL != "https://api.test/x" || parsed.Headers["X-Name"] != "it's me" || parsed.Body != `{"a":1}` {
		t.Fatalf("unexpected parse: %#v", parsed)
	}
	if parsed, err = curlparser.Parse(InitialCurl([]string{"https://api.test/y"})); err != nil || parsed.URL != "https://api.test/y" {
		t.Fatalf("url shortcut failed: %#v %v", parsed, err)
	}
}
