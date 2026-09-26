package requestutil

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// ResponseFile describes a response that is a downloadable file, not text.
type ResponseFile struct {
	Name string `json:"name"` // suggested file name
	Kind string `json:"kind"` // human label, e.g. "PDF" or "Excel"
}

// kinds maps media types to a readable label and a fallback extension.
var kinds = map[string][2]string{
	"application/pdf":    {"PDF", ".pdf"},
	"application/zip":    {"ZIP", ".zip"},
	"application/gzip":   {"GZIP", ".gz"},
	"application/msword": {"Word", ".doc"},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {"Word", ".docx"},
	"application/vnd.ms-excel": {"Excel", ".xls"},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {"Excel", ".xlsx"},
	"application/vnd.ms-powerpoint":                                             {"PowerPoint", ".ppt"},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": {"PowerPoint", ".pptx"},
	"text/csv":                 {"CSV", ".csv"},
	"application/octet-stream": {"Binary", ".bin"},
}

// DetectFile reports whether a response should be treated as a file:
// an attachment, a known document/archive/media type, or non-UTF-8 bytes.
func DetectFile(h http.Header, body string) (ResponseFile, bool) {
	mediaType, _, _ := mime.ParseMediaType(h.Get("Content-Type"))
	disposition, params, _ := mime.ParseMediaType(h.Get("Content-Disposition"))
	name := filepath.Base(params["filename"])
	if name == "." || name == "/" {
		name = ""
	}

	kind, known := kinds[mediaType]
	major, _, _ := strings.Cut(mediaType, "/")
	isFile := disposition == "attachment" || known ||
		major == "image" || major == "audio" || major == "video" ||
		(!utf8.ValidString(body) && body != "")
	if !isFile {
		return ResponseFile{}, false
	}

	label, ext := kind[0], kind[1]
	if label == "" {
		label = strings.ToUpper(major)
		if exts, _ := mime.ExtensionsByType(mediaType); len(exts) > 0 {
			ext = exts[0]
		}
		if label == "" {
			label = "File"
		}
	}
	if name == "" {
		name = "response-" + time.Now().Format("20060102-150405") + ext
	}
	return ResponseFile{Name: name, Kind: label}, true
}
