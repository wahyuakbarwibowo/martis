package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestVersionReadsRedirectTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/owner/repo/releases/tag/v1.2.3", http.StatusFound)
	}))
	defer srv.Close()
	got, err := latestVersion(srv.URL)
	if err != nil || got != "v1.2.3" {
		t.Fatalf("got %q, %v", got, err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer bad.Close()
	if _, err := latestVersion(bad.URL); err == nil {
		t.Fatal("expected error without a release redirect")
	}
}
