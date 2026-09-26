package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"martis/internal/domain"
	"martis/internal/environment"
	"martis/internal/requestutil"
)

// RunFolder sends every request in a collection folder in order and
// returns how many failed (transport error, HTTP >= 400, or assertion).
func RunFolder(col *domain.Collection, folder string, vars map[string]string, do func(domain.RequestPayload) domain.ResponseResult, out io.Writer) (int, error) {
	var items []domain.CollectionItem
	found := false
	for _, f := range col.Folders {
		if strings.EqualFold(f.Name, folder) {
			items, found = f.Items, true
			break
		}
	}
	if !found {
		return 0, fmt.Errorf("folder %q tidak ditemukan", folder)
	}
	if vars == nil {
		vars = map[string]string{}
	}
	failed := 0
	for _, it := range items {
		p := domain.RequestPayload{Method: it.Method, URL: it.URL, HeaderKey: it.HeaderKey, HeaderVal: it.HeaderVal, HeaderAuth: it.HeaderAuth, BodyType: it.BodyType, BodyRaw: it.BodyRaw, Variables: it.Variables, BodyFile: it.BodyFile, FormKey: it.FormKey, FormPath: it.FormPath, FormFields: it.FormFields, FormFiles: it.FormFiles, Headers: it.Headers, Auth: it.Auth, Assertions: it.Assertions}
		prepared, err := requestutil.Prepare(p, vars)
		var problems []string
		r := domain.ResponseResult{Err: err}
		if err == nil {
			r = do(prepared)
		}
		if r.Err != nil {
			problems = append(problems, r.Err.Error())
		} else {
			captured, captureFailures := requestutil.Capture(r, it.Assertions)
			for k, v := range captured {
				vars[k] = v
			}
			problems = append(requestutil.Assert(r, it.Assertions), captureFailures...)
			if r.StatusCode >= 400 {
				problems = append(problems, fmt.Sprintf("HTTP %d", r.StatusCode))
			}
		}
		mark := "✅"
		if len(problems) > 0 {
			mark = "❌"
			failed++
		}
		fmt.Fprintf(out, "%s [%-6s] %-25s %d %v\n", mark, it.Method, it.Name, r.StatusCode, r.Duration.Round(1e6))
		for _, problem := range problems {
			fmt.Fprintf(out, "     - %s\n", problem)
		}
	}
	fmt.Fprintf(out, "\n%d request, %d gagal\n", len(items), failed)
	return failed, nil
}

// loadEnv accepts "prod" or "prod.env" from the environments directory.
func loadEnv(dir, name string) (map[string]string, error) {
	if name == "" {
		return nil, nil
	}
	if !strings.HasSuffix(name, ".env") {
		name += ".env"
	}
	return environment.Load(filepath.Join(dir, filepath.Base(name)))
}
