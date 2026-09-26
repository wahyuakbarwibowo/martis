package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"martis/internal/domain"
)

// Client interface untuk abstraksi HTTP execution
type Client interface {
	Do(payload domain.RequestPayload) domain.ResponseResult
}

type defaultHTTPClient struct {
	userAgent string
}

// NewClient membuat instance HTTP Client baru
func NewClient(userAgent ...string) Client {
	ua := "Martis-TUI-Client/1.0"
	if len(userAgent) > 0 && userAgent[0] != "" {
		ua = userAgent[0]
	}
	return &defaultHTTPClient{userAgent: ua}
}

// buildRequest turns a payload into an *http.Request (auth, body, headers).
func (c *defaultHTTPClient) buildRequest(payload domain.RequestPayload) (*http.Request, error) {
	if payload.Auth.Mode == "oauth2" && payload.Auth.Token == "" {
		token, err := OAuthToken(payload.Auth)
		if err != nil {
			return nil, err
		}
		payload.HeaderAuth = "Bearer " + token
	}
	targetURL := strings.TrimSpace(payload.URL)
	if targetURL == "" {
		targetURL = "https://httpbin.org/anything"
	}
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	method := strings.ToUpper(strings.TrimSpace(payload.Method))
	if method == "" {
		method = "GET"
	}

	var reqBody io.Reader
	var contentType string

	if method != "GET" && method != "HEAD" {
		if payload.BodyType == "raw" {
			if payload.BodyFile != "" {
				data, err := os.ReadFile(payload.BodyFile)
				if err != nil {
					return nil, fmt.Errorf("read body file error: %w", err)
				}
				reqBody = bytes.NewReader(data)
			} else {
				reqBody = bytes.NewBufferString(payload.BodyRaw)
			}
		} else if payload.BodyType == "form" && (payload.FormPath != "" || len(payload.FormFiles) > 0 || len(payload.FormFields) > 0) {
			bodyBuf := &bytes.Buffer{}
			writer := multipart.NewWriter(bodyBuf)
			for _, field := range payload.FormFields {
				if strings.TrimSpace(field.Key) != "" {
					if err := writer.WriteField(field.Key, field.Value); err != nil {
						return nil, fmt.Errorf("write form field error: %w", err)
					}
				}
			}
			files := append([]domain.KeyValue(nil), payload.FormFiles...)
			if payload.FormPath != "" {
				key := strings.TrimSpace(payload.FormKey)
				if key == "" {
					key = "file"
				}
				files = append(files, domain.KeyValue{Key: key, Value: payload.FormPath})
			}
			for _, field := range files {
				file, err := os.Open(field.Value)
				if err != nil {
					return nil, fmt.Errorf("open file error: %w", err)
				}
				part, err := writer.CreateFormFile(field.Key, filepath.Base(field.Value))
				if err == nil {
					_, err = io.Copy(part, file)
				}
				_ = file.Close()
				if err != nil {
					return nil, fmt.Errorf("write form file error: %w", err)
				}
			}

			_ = writer.Close()
			contentType = writer.FormDataContentType()
			reqBody = bodyBuf
		}
	}

	req, err := http.NewRequest(method, targetURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Apply Headers
	for _, row := range payload.Headers {
		if row.Key != "" {
			req.Header.Add(row.Key, row.Value)
		}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if payload.HeaderKey != "" {
		req.Header.Set(payload.HeaderKey, payload.HeaderVal)
	}
	if payload.HeaderAuth != "" {
		req.Header.Set("Authorization", payload.HeaderAuth)
	}
	req.Header.Set("User-Agent", c.userAgent)
	return req, nil
}

// Do mengeksekusi request HTTP dan mengembalikan struct response bersih
func (c *defaultHTTPClient) Do(payload domain.RequestPayload) domain.ResponseResult {
	startTime := time.Now()
	req, err := c.buildRequest(payload)
	if err != nil {
		return domain.ResponseResult{Err: err, Duration: time.Since(startTime)}
	}

	timeout := payload.TimeoutSecs
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	duration := time.Since(startTime)
	if err != nil {
		return domain.ResponseResult{
			Err:      err,
			Duration: duration,
		}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.ResponseResult{
			StatusCode: resp.StatusCode,
			StatusText: resp.Status,
			Proto:      resp.Proto,
			Duration:   duration,
			Headers:    resp.Header,
			Err:        fmt.Errorf("reading response: %w", err),
		}
	}

	duration = time.Since(startTime)
	// Pretty format JSON if valid
	bodyStr := string(respBytes)
	var prettyJSON bytes.Buffer
	if json.Indent(&prettyJSON, respBytes, "", "  ") == nil {
		bodyStr = prettyJSON.String()
	}

	return domain.ResponseResult{
		StatusCode: resp.StatusCode,
		StatusText: resp.Status,
		Proto:      resp.Proto,
		Duration:   duration,
		Headers:    resp.Header,
		Body:       bodyStr,
	}
}

// OAuthToken obtains an access token using OAuth 2.0 client credentials.
func OAuthToken(auth domain.AuthConfig) (string, error) {
	u, err := url.Parse(auth.TokenURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))) {
		return "", fmt.Errorf("token URL must use HTTPS (HTTP allowed on localhost)")
	}
	values := url.Values{"grant_type": {"client_credentials"}}
	if auth.Scope != "" {
		values.Set("scope", auth.Scope)
	}
	req, err := http.NewRequest("POST", u.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(auth.Username, auth.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}
	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&token); err != nil {
		return "", fmt.Errorf("invalid token response: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("token endpoint returned no access_token")
	}
	if token.TokenType != "" && !strings.EqualFold(token.TokenType, "bearer") {
		return "", fmt.Errorf("unsupported token type")
	}
	return token.AccessToken, nil
}
