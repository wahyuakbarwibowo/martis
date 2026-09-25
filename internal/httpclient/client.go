package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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

// Do mengeksekusi request HTTP dan mengembalikan struct response bersih
func (c *defaultHTTPClient) Do(payload domain.RequestPayload) domain.ResponseResult {
	startTime := time.Now()

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
			reqBody = bytes.NewBufferString(payload.BodyRaw)
		} else if payload.BodyType == "form" && payload.FormPath != "" {
			bodyBuf := &bytes.Buffer{}
			writer := multipart.NewWriter(bodyBuf)

			file, err := os.Open(payload.FormPath)
			if err != nil {
				return domain.ResponseResult{
					Err:      fmt.Errorf("open file error: %w", err),
					Duration: time.Since(startTime),
				}
			}
			defer file.Close()

			fieldName := strings.TrimSpace(payload.FormKey)
			if fieldName == "" {
				fieldName = "file"
			}

			part, err := writer.CreateFormFile(fieldName, filepath.Base(payload.FormPath))
			if err != nil {
				return domain.ResponseResult{
					Err:      fmt.Errorf("create form file error: %w", err),
					Duration: time.Since(startTime),
				}
			}
			if _, err = io.Copy(part, file); err != nil {
				return domain.ResponseResult{
					Err:      fmt.Errorf("copy file data error: %w", err),
					Duration: time.Since(startTime),
				}
			}

			_ = writer.Close()
			contentType = writer.FormDataContentType()
			reqBody = bodyBuf
		}
	}

	req, err := http.NewRequest(method, targetURL, reqBody)
	if err != nil {
		return domain.ResponseResult{
			Err:      fmt.Errorf("invalid request: %w", err),
			Duration: time.Since(startTime),
		}
	}

	// Apply Headers
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else if payload.HeaderKey != "" && payload.HeaderVal != "" {
		req.Header.Set(payload.HeaderKey, payload.HeaderVal)
	}
	if payload.HeaderAuth != "" {
		req.Header.Set("Authorization", payload.HeaderAuth)
	}
	req.Header.Set("User-Agent", c.userAgent)

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
