package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"martis/internal/domain"
)

// File reads a Postman v2 collection or OpenAPI 3 document.
func File(path string) (*domain.Collection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode import: %w", err)
	}
	if _, ok := raw["openapi"]; ok {
		return openAPI(raw, filepath.Base(path))
	}
	if _, ok := raw["info"]; ok {
		return postman(raw, filepath.Base(path))
	}
	return nil, fmt.Errorf("unsupported collection format")
}

func postman(raw map[string]any, name string) (*domain.Collection, error) {
	col := &domain.Collection{Name: name}
	if info, ok := raw["info"].(map[string]any); ok {
		if s, ok := info["name"].(string); ok && s != "" {
			col.Name = s
		}
	}
	folder := func(name string) *domain.Folder {
		for i := range col.Folders {
			if col.Folders[i].Name == name {
				return &col.Folders[i]
			}
		}
		col.Folders = append(col.Folders, domain.Folder{ID: id(), Name: name, IsExpanded: true})
		return &col.Folders[len(col.Folders)-1]
	}
	// cur is nil for requests at the top level; they go to "Imported".
	var walk func([]any, *[]domain.Folder, *domain.Folder)
	walk = func(items []any, list *[]domain.Folder, cur *domain.Folder) {
		for _, value := range items {
			item, ok := value.(map[string]any)
			if !ok {
				continue
			}
			if children, ok := item["item"].([]any); ok {
				*list = append(*list, domain.Folder{ID: id(), Name: stringValue(item["name"], "Folder"), IsExpanded: true})
				child := &(*list)[len(*list)-1]
				walk(children, &child.Folders, child)
				continue
			}
			request, ok := item["request"].(map[string]any)
			if !ok {
				continue
			}
			ci := domain.CollectionItem{ID: id(), Name: stringValue(item["name"], "Imported request"), Method: strings.ToUpper(stringValue(request["method"], "GET")), BodyType: "raw"}
			if u, ok := request["url"].(map[string]any); ok {
				ci.URL = stringValue(u["raw"], "")
			} else {
				ci.URL = stringValue(request["url"], "")
			}
			if hs, ok := request["header"].([]any); ok {
				for _, h := range hs {
					if x, ok := h.(map[string]any); ok {
						ci.Headers = append(ci.Headers, domain.KeyValue{Key: stringValue(x["key"], ""), Value: stringValue(x["value"], "")})
					}
				}
			}
			if body, ok := request["body"].(map[string]any); ok {
				mode := stringValue(body["mode"], "")
				if mode == "raw" {
					ci.BodyRaw = stringValue(body["raw"], "")
				} else if mode == "formdata" {
					ci.BodyType = "form"
					for _, f := range anySlice(body["formdata"]) {
						if x, ok := f.(map[string]any); ok {
							key := stringValue(x["key"], "")
							if stringValue(x["type"], "text") == "file" {
								ci.FormFiles = append(ci.FormFiles, domain.KeyValue{Key: key, Value: stringValue(x["src"], "")})
							} else {
								ci.FormFields = append(ci.FormFields, domain.KeyValue{Key: key, Value: stringValue(x["value"], "")})
							}
						}
					}
				}
			}
			f := cur
			if f == nil {
				f = folder("Imported")
			}
			f.Items = append(f.Items, ci)
		}
	}
	walk(anySlice(raw["item"]), &col.Folders, nil)
	return col, nil
}

func openAPI(raw map[string]any, name string) (*domain.Collection, error) {
	col := &domain.Collection{Name: name}
	if info, ok := raw["info"].(map[string]any); ok {
		if s, ok := info["title"].(string); ok && s != "" {
			col.Name = s
		}
	}
	server := ""
	if ss := anySlice(raw["servers"]); len(ss) > 0 {
		if x, ok := ss[0].(map[string]any); ok {
			server = stringValue(x["url"], "")
		}
	}
	paths, ok := raw["paths"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("OpenAPI paths not found")
	}
	folder := &domain.Folder{ID: id(), Name: "OpenAPI", IsExpanded: true}
	for path, value := range paths {
		methods, _ := value.(map[string]any)
		for method, opValue := range methods {
			if !map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true, "head": true, "options": true}[strings.ToLower(method)] {
				continue
			}
			op, _ := opValue.(map[string]any)
			ci := domain.CollectionItem{ID: id(), Name: stringValue(op["summary"], strings.ToUpper(method)+" "+path), Method: strings.ToUpper(method), URL: strings.TrimRight(server, "/") + path, BodyType: "raw"}
			if req, ok := op["requestBody"].(map[string]any); ok {
				if content, ok := req["content"].(map[string]any); ok {
					if app, ok := content["application/json"].(map[string]any); ok {
						if ex, ok := app["example"]; ok {
							b, _ := json.MarshalIndent(ex, "", "  ")
							ci.BodyRaw = string(b)
						}
					}
				}
			}
			folder.Items = append(folder.Items, ci)
		}
	}
	col.Folders = append(col.Folders, *folder)
	return col, nil
}

func anySlice(v any) []any {
	if x, ok := v.([]any); ok {
		return x
	}
	return nil
}
func stringValue(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

var idSeq atomic.Uint64

// id stays unique even when several calls share the same clock tick.
func id() string { return fmt.Sprintf("import-%d-%d", time.Now().UnixNano(), idSeq.Add(1)) }
