package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"martis/internal/domain"
)

func TestStreamDeliversEventsAsTheyArrive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 1; i <= 3; i++ {
			fmt.Fprintf(w, "event: tick\ndata: %d\n\n", i)
			w.(http.Flusher).Flush()
			time.Sleep(20 * time.Millisecond)
		}
	}))
	defer srv.Close()

	var events []string
	res := Stream(context.Background(), NewClient(), domain.RequestPayload{Method: "GET", URL: srv.URL}, func(e string) { events = append(events, e) })
	if res.Err != nil || res.StatusCode != 200 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(events) != 3 || events[2] != "event: tick\ndata: 3\n" {
		t.Fatalf("events: %q", events)
	}
	if !strings.Contains(res.Body, "data: 2") {
		t.Fatalf("body missing events: %q", res.Body)
	}
}

func TestStreamStopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done() // never ends on its own
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan domain.ResponseResult)
	go func() {
		done <- Stream(ctx, NewClient(), domain.RequestPayload{Method: "GET", URL: srv.URL}, func(string) { cancel() })
	}()
	select {
	case res := <-done:
		if res.Err != nil || !strings.Contains(res.Body, "data: first") {
			t.Fatalf("unexpected result after cancel: %+v", res)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("stream did not stop after cancel")
	}
}

func TestStreamReadsNormalResponsesWhole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()
	called := false
	res := Stream(context.Background(), NewClient(), domain.RequestPayload{Method: "GET", URL: srv.URL}, func(string) { called = true })
	if called || res.Body != "{\n  \"ok\": true\n}" {
		t.Fatalf("called=%v body=%q", called, res.Body)
	}
}
