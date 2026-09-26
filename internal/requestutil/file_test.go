package requestutil

import (
	"net/http"
	"strings"
	"testing"
)

func TestDetectFile(t *testing.T) {
	cases := []struct {
		name, contentType, disposition, body string
		isFile                               bool
		wantName, wantKind                   string
	}{
		{"json is text", "application/json", "", `{"a":1}`, false, "", ""},
		{"html is text", "text/html; charset=utf-8", "", "<p>hi</p>", false, "", ""},
		{"pdf by type", "application/pdf", "", "%PDF-1.7", true, ".pdf", "PDF"},
		{"excel with filename", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", `attachment; filename="laporan Q3.xlsx"`, "PK\x03\x04", true, "laporan Q3.xlsx", "Excel"},
		{"attachment csv", "text/csv", `attachment; filename=data.csv`, "a,b\n1,2", true, "data.csv", "CSV"},
		{"image", "image/png", "", "\x89PNG", true, ".png", "IMAGE"},
		{"binary without type", "", "", "\xff\xfe\x00\x01", true, "", "File"},
		{"path in filename is stripped", "application/pdf", `attachment; filename="../../etc/passwd"`, "x", true, "passwd", "PDF"},
	}
	for _, c := range cases {
		h := http.Header{}
		if c.contentType != "" {
			h.Set("Content-Type", c.contentType)
		}
		if c.disposition != "" {
			h.Set("Content-Disposition", c.disposition)
		}
		f, ok := DetectFile(h, c.body)
		if ok != c.isFile {
			t.Fatalf("%s: isFile=%v", c.name, ok)
		}
		if !ok {
			continue
		}
		if f.Kind != c.wantKind || (c.wantName != "" && !strings.HasSuffix(f.Name, c.wantName)) {
			t.Fatalf("%s: got %+v", c.name, f)
		}
	}
}
