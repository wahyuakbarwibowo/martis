package cli

import (
	"io"
	"testing"

	"martis/internal/domain"
)

func TestRunFolderChainsCapturedVariables(t *testing.T) {
	col := &domain.Collection{Folders: []domain.Folder{{Name: "Auth", Items: []domain.CollectionItem{
		{Name: "login", Method: "POST", URL: "{{base}}/login", Assertions: "Status == 200\nset token = json.token"},
		{Name: "me", Method: "GET", URL: "{{base}}/me", HeaderAuth: "Bearer {{token}}", Assertions: "Status == 200"},
		{Name: "broken", Method: "GET", URL: "{{base}}/missing"},
	}}}}
	var seen []string
	do := func(p domain.RequestPayload) domain.ResponseResult {
		seen = append(seen, p.URL+" "+p.HeaderAuth)
		switch p.URL {
		case "https://api.test/login":
			return domain.ResponseResult{StatusCode: 200, Body: `{"token":"abc"}`}
		case "https://api.test/me":
			return domain.ResponseResult{StatusCode: 200}
		}
		return domain.ResponseResult{StatusCode: 404}
	}
	failed, err := RunFolder(col, "auth", map[string]string{"base": "https://api.test"}, do, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if failed != 1 || seen[1] != "https://api.test/me Bearer abc" {
		t.Fatalf("failed=%d seen=%v", failed, seen)
	}
	if _, err := RunFolder(col, "nope", nil, do, io.Discard); err == nil {
		t.Fatal("expected missing folder error")
	}
}
