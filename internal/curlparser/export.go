package curlparser

import (
	"encoding/base64"
	"martis/internal/domain"
	"net/url"
	"strings"
)

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// Export produces a POSIX-shell command without executing it.
func Export(p domain.RequestPayload) string {
	method := strings.ToUpper(p.Method)
	if method == "" {
		method = "GET"
	}
	if p.Auth.Mode == "bearer" && p.HeaderAuth == "" {
		p.HeaderAuth = "Bearer " + p.Auth.Token
	}
	if p.Auth.Mode == "basic" && p.HeaderAuth == "" {
		p.HeaderAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(p.Auth.Username+":"+p.Auth.Password))
	}
	if p.Auth.Mode == "api-key" && p.Auth.Location == "query" {
		u, err := url.Parse(p.URL)
		if err == nil {
			q := u.Query()
			q.Set(p.Auth.Key, p.Auth.Value)
			u.RawQuery = q.Encode()
			p.URL = u.String()
		}
	} else if p.Auth.Mode == "api-key" && p.Auth.Location != "" && p.Auth.Key != "" {
		p.Headers = append(p.Headers, domain.KeyValue{Key: p.Auth.Key, Value: p.Auth.Value})
	}
	parts := []string{"curl", "-X", quote(method), "--url", quote(p.URL)}
	for _, h := range p.Headers {
		if h.Key != "" {
			parts = append(parts, "-H", quote(h.Key+": "+h.Value))
		}
	}
	if p.HeaderKey != "" {
		parts = append(parts, "-H", quote(p.HeaderKey+": "+p.HeaderVal))
	}
	if p.HeaderAuth != "" {
		parts = append(parts, "-H", quote("Authorization: "+p.HeaderAuth))
	}
	if p.BodyType == "form" && p.FormPath != "" {
		parts = append(parts, "-F", quote(p.FormKey+"=@"+p.FormPath))
	} else if p.BodyRaw != "" && method != "GET" && method != "HEAD" {
		parts = append(parts, "--data-raw", quote(p.BodyRaw))
	}
	return strings.Join(parts, " ")
}
