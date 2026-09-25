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
	fields := []*string{&p.URL, &p.BodyRaw, &p.HeaderKey, &p.HeaderVal, &p.HeaderAuth, &p.FormKey, &p.FormPath, &p.Auth.Token, &p.Auth.Username, &p.Auth.Password, &p.Auth.Key, &p.Auth.Value, &p.Auth.TokenURL, &p.Auth.Scope}
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

func Query(raw string) ([]domain.KeyValue, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}
	var rows []domain.KeyValue
	// Preserve URL order, including duplicate keys.
	for _, pair := range strings.Split(u.RawQuery, "&") {
		if pair == "" {
			continue
		}
		k, _, _ := strings.Cut(pair, "=")
		key, _ := url.QueryUnescape(k)
		if len(q[key]) > 0 {
			rows = append(rows, domain.KeyValue{Key: key, Value: q[key][0]})
			q[key] = q[key][1:]
		}
	}
	return rows, nil
}

func WithQuery(raw string, rows []domain.KeyValue) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	var pairs []string
	for _, row := range rows {
		if row.Key != "" {
			pairs = append(pairs, url.QueryEscape(row.Key)+"="+url.QueryEscape(row.Value))
		}
	}
	u.RawQuery = strings.Join(pairs, "&")
	return u.String(), nil
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
			var value any
			if json.Unmarshal([]byte(r.Body), &value) != nil {
				failures = append(failures, line)
				continue
			}
			path := strings.TrimSuffix(strings.TrimPrefix(line, "json."), " != nil")
			for _, key := range strings.Split(path, ".") {
				object, ok := value.(map[string]any)
				if !ok {
					value = nil
					break
				}
				value = object[key]
			}
			if value == nil {
				failures = append(failures, line)
			}
		} else {
			failures = append(failures, "unsupported assertion: "+line)
		}
	}
	return failures
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
