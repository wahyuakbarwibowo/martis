package requestutil

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"martis/internal/domain"
	"martis/internal/environment"
)

func Prepare(p domain.RequestPayload, vars map[string]string) (domain.RequestPayload, error) {
	p.Headers = append([]domain.KeyValue(nil), p.Headers...)
	fields := []*string{&p.URL, &p.BodyRaw, &p.Variables, &p.HeaderKey, &p.HeaderVal, &p.HeaderAuth, &p.FormKey, &p.FormPath, &p.Auth.Token, &p.Auth.Username, &p.Auth.Password, &p.Auth.Key, &p.Auth.Value, &p.Auth.TokenURL, &p.Auth.Scope}
	for i := range p.Headers {
		fields = append(fields, &p.Headers[i].Key, &p.Headers[i].Value)
	}
	for _, field := range fields {
		value, err := environment.Expand(*field, vars)
		if err != nil {
			return p, err
		}
		*field = value
	}
	if p.BodyType == "graphql" {
		if err := graphQLBody(&p); err != nil {
			return p, err
		}
	}
	switch p.Auth.Mode {
	case "", "none":
	case "bearer", "oauth2":
		p.HeaderAuth = "Bearer " + p.Auth.Token
	case "basic":
		p.HeaderAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(p.Auth.Username+":"+p.Auth.Password))
	case "api-key":
		if p.Auth.Key == "" {
			return p, fmt.Errorf("API key name is required")
		}
		if p.Auth.Location == "query" {
			u, err := url.Parse(p.URL)
			if err != nil {
				return p, err
			}
			q := u.Query()
			q.Set(p.Auth.Key, p.Auth.Value)
			u.RawQuery = q.Encode()
			p.URL = u.String()
		} else {
			p.Headers = append(p.Headers, domain.KeyValue{Key: p.Auth.Key, Value: p.Auth.Value})
		}
	default:
		return p, fmt.Errorf("unknown authentication mode %q", p.Auth.Mode)
	}
	return p, nil
}

// graphQLBody turns a GraphQL query plus JSON variables into a JSON POST body.
func graphQLBody(p *domain.RequestPayload) error {
	body := map[string]any{"query": p.BodyRaw}
	if vars := strings.TrimSpace(p.Variables); vars != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(vars), &parsed); err != nil {
			return fmt.Errorf("GraphQL variables harus berupa objek JSON: %w", err)
		}
		body["variables"] = parsed
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	p.BodyType, p.BodyRaw, p.BodyFile = "raw", string(data), ""
	if strings.EqualFold(p.HeaderKey, "Content-Type") {
		p.HeaderVal = "application/json"
		return nil
	}
	for i := range p.Headers {
		if strings.EqualFold(p.Headers[i].Key, "Content-Type") {
			p.Headers[i].Value = "application/json"
			return nil
		}
	}
	p.Headers = append(p.Headers, domain.KeyValue{Key: "Content-Type", Value: "application/json"})
	return nil
}

// splitURL splits raw into base, query and fragment without url.Parse, so
// {{vars}} and malformed escapes survive untouched.
func splitURL(raw string) (base, query, frag string) {
	raw, frag, hasFrag := strings.Cut(raw, "#")
	if hasFrag {
		frag = "#" + frag
	}
	base, query, _ = strings.Cut(raw, "?")
	return base, query, frag
}

// unescape decodes s, falling back to the raw text on invalid escapes.
func unescape(s string) string {
	if v, err := url.QueryUnescape(s); err == nil {
		return v
	}
	return s
}

// escape query-escapes s but keeps {{var}} placeholders readable.
func escape(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "{{")
		j := strings.Index(s[max(i, 0):], "}}")
		if i < 0 || j < 0 {
			return b.String() + url.QueryEscape(s)
		}
		j += i + 2
		b.WriteString(url.QueryEscape(s[:i]) + s[i:j])
		s = s[j:]
	}
}

// Query returns the URL's query params in order, including duplicate keys.
func Query(raw string) ([]domain.KeyValue, error) {
	_, query, _ := splitURL(raw)
	var rows []domain.KeyValue
	for _, pair := range strings.Split(query, "&") {
		if pair == "" {
			continue
		}
		k, v, _ := strings.Cut(pair, "=")
		rows = append(rows, domain.KeyValue{Key: unescape(k), Value: unescape(v)})
	}
	return rows, nil
}

func WithQuery(raw string, rows []domain.KeyValue) (string, error) {
	base, _, frag := splitURL(raw)
	var pairs []string
	for _, row := range rows {
		if row.Key != "" {
			pairs = append(pairs, escape(row.Key)+"="+escape(row.Value))
		}
	}
	if len(pairs) > 0 {
		base += "?" + strings.Join(pairs, "&")
	}
	return base + frag, nil
}

// Assert supports Status == N and json.path != nil, one expression per line.
func Assert(r domain.ResponseResult, script string) []string {
	var failures []string
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Status == ") {
			code, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Status == ")))
			if err != nil || r.StatusCode != code {
				failures = append(failures, line)
			}
		} else if strings.HasPrefix(line, "json.") && strings.HasSuffix(line, " != nil") {
			if jsonPath(r.Body, strings.TrimSuffix(line, " != nil")) == nil {
				failures = append(failures, line)
			}
		} else if strings.HasPrefix(line, "set ") {
			// Handled by Capture.
		} else {
			failures = append(failures, "unsupported assertion: "+line)
		}
	}
	return failures
}

// jsonPath resolves "json.a.0.b" against a JSON body; nil when missing.
func jsonPath(body, path string) any {
	var value any
	if json.Unmarshal([]byte(body), &value) != nil {
		return nil
	}
	for _, key := range strings.Split(strings.TrimPrefix(path, "json."), ".") {
		switch v := value.(type) {
		case map[string]any:
			value = v[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(v) {
				return nil
			}
			value = v[i]
		default:
			return nil
		}
	}
	return value
}

// JSONPath returns the pretty-printed value at "json.a.0.b", or false when missing.
func JSONPath(body, path string) (string, bool) {
	value := jsonPath(body, path)
	if value == nil {
		return "", false
	}
	b, _ := json.MarshalIndent(value, "", "  ")
	return string(b), true
}

// Capture reads "set name = json.path" lines and returns the captured values.
func Capture(r domain.ResponseResult, script string) (map[string]string, []string) {
	values := map[string]string{}
	var failures []string
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(line, "set ")
		if !ok {
			continue
		}
		name, path, ok := strings.Cut(rest, "=")
		name, path = strings.TrimSpace(name), strings.TrimSpace(path)
		if !ok || !strings.HasPrefix(path, "json.") || !environment.ValidName(name) {
			failures = append(failures, "invalid capture: "+line)
			continue
		}
		switch v := jsonPath(r.Body, path).(type) {
		case nil:
			failures = append(failures, line)
		case string:
			values[name] = v
		default:
			b, _ := json.Marshal(v)
			values[name] = string(b)
		}
	}
	return values, failures
}

type BenchmarkResult struct {
	Count, Failed     int
	Average, Min, Max time.Duration
}

func Benchmark(do func(domain.RequestPayload) domain.ResponseResult, p domain.RequestPayload, count int) (BenchmarkResult, error) {
	if count < 1 || count > 1000 {
		return BenchmarkResult{}, fmt.Errorf("request count must be 1..1000")
	}
	result := BenchmarkResult{Count: count}
	var total time.Duration
	for i := 0; i < count; i++ {
		r := do(p)
		total += r.Duration
		if r.Err != nil || r.StatusCode >= 400 || len(Assert(r, p.Assertions)) > 0 {
			result.Failed++
		}
		if i == 0 || r.Duration < result.Min {
			result.Min = r.Duration
		}
		if r.Duration > result.Max {
			result.Max = r.Duration
		}
	}
	result.Average = total / time.Duration(count)
	return result, nil
}
