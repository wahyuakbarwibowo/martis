package domain

import (
	"fmt"
	"net/http"
	"time"
)

// HTTPMethod merepresentasikan metode HTTP yang valid
type HTTPMethod string

const (
	MethodGET    HTTPMethod = "GET"
	MethodPOST   HTTPMethod = "POST"
	MethodPUT    HTTPMethod = "PUT"
	MethodDELETE HTTPMethod = "DELETE"
	MethodPATCH  HTTPMethod = "PATCH"
	MethodHEAD   HTTPMethod = "HEAD"
)

// SupportedMethods mengembalikan daftar method umum
func SupportedMethods() []string {
	return []string{
		string(MethodGET),
		string(MethodPOST),
		string(MethodPUT),
		string(MethodDELETE),
		string(MethodPATCH),
		string(MethodHEAD),
	}
}

// RequestPayload merepresentasikan data request yang akan dikirim
type RequestPayload struct {
	Headers     []KeyValue
	Auth        AuthConfig
	Assertions  string
	Method      string
	URL         string
	HeaderKey   string
	HeaderVal   string
	HeaderAuth  string
	BodyType    string // "raw", "form", atau "graphql" (BodyRaw berisi query)
	BodyRaw     string
	Variables   string // variables GraphQL dalam JSON
	BodyFile    string
	FormKey     string
	FormPath    string
	FormFields  []KeyValue
	FormFiles   []KeyValue
	TimeoutSecs time.Duration
}

// ResponseResult merepresentasikan hasil eksekusi HTTP
type ResponseResult struct {
	StatusCode int
	StatusText string
	Proto      string
	Duration   time.Duration
	Headers    http.Header
	Body       string
	Err        error
}

// CollectionItem merepresentasikan item tersimpan dalam sebuah folder collection
type CollectionItem struct {
	Headers    []KeyValue `json:"headers,omitempty"`
	Auth       AuthConfig `json:"auth,omitempty"`
	Assertions string     `json:"assertions,omitempty"`
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Method     string     `json:"method"`
	URL        string     `json:"url"`
	HeaderKey  string     `json:"header_key,omitempty"`
	HeaderVal  string     `json:"header_val,omitempty"`
	HeaderAuth string     `json:"header_auth,omitempty"`
	BodyType   string     `json:"body_type"` // "raw", "form", atau "graphql"
	BodyRaw    string     `json:"body_raw,omitempty"`
	Variables  string     `json:"variables,omitempty"`
	BodyFile   string     `json:"body_file,omitempty"`
	FormKey    string     `json:"form_key,omitempty"`
	FormPath   string     `json:"form_path,omitempty"`
	FormFields []KeyValue `json:"form_fields,omitempty"`
	FormFiles  []KeyValue `json:"form_files,omitempty"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// OAuth helper supports the client-credentials grant.
type AuthConfig struct {
	Mode     string `json:"mode,omitempty"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
	Location string `json:"location,omitempty"`
	TokenURL string `json:"token_url,omitempty"`
	Scope    string `json:"scope,omitempty"`
}

type HistoryEntry struct {
	At      time.Time      `json:"at"`
	Request CollectionItem `json:"request"`
	Status  int            `json:"status"`
}

// Folder merepresentasikan sebuah grup dalam collection
type Folder struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	IsExpanded bool             `json:"is_expanded"`
	Items      []CollectionItem `json:"items"`
	Folders    []Folder         `json:"folders,omitempty"`
}

// FolderRef menunjuk satu folder di pohon collection beserta posisinya.
type FolderRef struct {
	*Folder
	Depth  int
	Path   string // nama lengkap, contoh "API / Users"
	parent *[]Folder
	index  int
}

// Remove menghapus folder ini (beserta isinya) dari induknya.
func (r FolderRef) Remove() {
	*r.parent = append((*r.parent)[:r.index], (*r.parent)[r.index+1:]...)
}

// FlatFolders mengembalikan semua folder secara depth-first (pre-order).
// Pointer valid sampai slice Folders mana pun diubah.
func (c *Collection) FlatFolders() []FolderRef {
	var out []FolderRef
	var walk func(*[]Folder, int, string)
	walk = func(list *[]Folder, depth int, prefix string) {
		for i := range *list {
			f := &(*list)[i]
			ref := FolderRef{Folder: f, Depth: depth, Path: prefix + f.Name, parent: list, index: i}
			out = append(out, ref)
			walk(&f.Folders, depth+1, ref.Path+" / ")
		}
	}
	walk(&c.Folders, 0, "")
	return out
}

// Collection merepresentasikan workspace kumpulan folder
type Collection struct {
	Name    string   `json:"name"`
	Folders []Folder `json:"folders"`
}

// FormatBytes mengonversi jumlah bytes ke representasi terbaca manusia (B, KB, MB, dsb)
func FormatBytes(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
