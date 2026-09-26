package httpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"martis/internal/domain"
)

// streamTransport keeps Do's 30s limit for the server's first answer, while an
// open event stream may run as long as the caller wants.
var streamTransport = func() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = 30 * time.Second
	return t
}()

// maxStreamBody caps how much of an event stream is kept in memory; older text is dropped.
const maxStreamBody = 1 << 20

// IsEventStream reports whether a response is Server-Sent Events.
func IsEventStream(h http.Header) bool {
	return strings.HasPrefix(strings.ToLower(h.Get("Content-Type")), "text/event-stream")
}

// Stream sends the request like Do, but when the server answers with
// text/event-stream it calls onEvent for every event as it arrives and keeps
// reading until the server closes the stream or ctx is cancelled. Other
// responses are read whole, exactly like Do.
func Stream(ctx context.Context, c Client, payload domain.RequestPayload, onEvent func(string)) domain.ResponseResult {
	dc, ok := c.(*defaultHTTPClient)
	if !ok {
		return c.Do(payload)
	}
	start := time.Now()
	req, err := dc.buildRequest(payload)
	if err != nil {
		return domain.ResponseResult{Err: err, Duration: time.Since(start)}
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "text/event-stream, */*")

	resp, err := (&http.Client{Transport: streamTransport}).Do(req)
	if err != nil {
		return domain.ResponseResult{Err: err, Duration: time.Since(start)}
	}
	defer resp.Body.Close()
	result := domain.ResponseResult{StatusCode: resp.StatusCode, StatusText: resp.Status, Proto: resp.Proto, Headers: resp.Header}

	if !IsEventStream(resp.Header) {
		body, err := io.ReadAll(resp.Body)
		var pretty bytes.Buffer
		if json.Indent(&pretty, body, "", "  ") == nil {
			body = pretty.Bytes()
		}
		result.Duration, result.Body, result.Err = time.Since(start), string(body), err
		return result
	}

	var body strings.Builder
	var event strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), maxStreamBody)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			event.WriteString(line + "\n")
			continue
		}
		// A blank line ends one event.
		if event.Len() == 0 {
			continue
		}
		text := event.String()
		event.Reset()
		if onEvent != nil {
			onEvent(text)
		}
		body.WriteString(text + "\n")
		if body.Len() > maxStreamBody {
			kept := body.String()[body.Len()-maxStreamBody/2:]
			body.Reset()
			body.WriteString(kept)
		}
	}
	if event.Len() > 0 { // stream closed without a trailing blank line
		if onEvent != nil {
			onEvent(event.String())
		}
		body.WriteString(event.String())
	}
	result.Duration, result.Body = time.Since(start), body.String()
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		result.Err = err
	}
	return result
}
