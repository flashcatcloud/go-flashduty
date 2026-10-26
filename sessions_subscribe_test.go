package flashduty

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const sseBody = ": ping\n\n" +
	`data: {"event_id":"e1","created_at":1000,"type":"assistant","invocation_id":"i1","root_invocation_id":"r1","payload":{"parts":[]}}` + "\n\n" +
	": ping\n\n" +
	"data: not-json\n\n" +
	`data: {"created_at":1001,"type":"turn_complete","payload":{}}` + "\n\n" +
	"data: [DONE]\n\n"

func TestSessionFrameReaderDecodesFramesAndStopsAtDone(t *testing.T) {
	fr := NewSessionFrameReader(strings.NewReader(sseBody))

	f1, err := fr.Next()
	if err != nil || f1.EventID != "e1" || f1.Type != "assistant" || f1.RootInvocationID != "r1" || f1.CreatedAt != 1000 {
		t.Fatalf("frame 1 = %+v, err = %v", f1, err)
	}
	f2, err := fr.Next()
	if err != nil || f2.Type != "turn_complete" || f2.EventID != "" {
		t.Fatalf("frame 2 = %+v, err = %v", f2, err)
	}
	if _, err := fr.Next(); err != io.EOF {
		t.Fatalf("after [DONE] err = %v, want io.EOF", err)
	}
}

func TestSessionFrameReaderCleanCloseIsEOF(t *testing.T) {
	fr := NewSessionFrameReader(strings.NewReader(`data: {"type":"assistant","payload":{}}` + "\n\n"))
	if _, err := fr.Next(); err != nil {
		t.Fatal(err)
	}
	if _, err := fr.Next(); err != io.EOF {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestSessionFrameReaderErrorEvent(t *testing.T) {
	fr := NewSessionFrameReader(strings.NewReader("event: error\ndata: \"stream broke\"\n\n"))
	_, err := fr.Next()
	var se *SessionStreamError
	if !errors.As(err, &se) || se.Message != "stream broke" {
		t.Fatalf("err = %v", err)
	}
}

func TestSessionsSubscribeStreamsFrames(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/safari/session/subscribe" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if acc := r.Header.Get("Accept"); acc != "text/event-stream" {
			t.Errorf("Accept = %q", acc)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"session_id":"s1"`) || !strings.Contains(string(body), `"last_event_at":42`) {
			t.Errorf("body = %s", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sseBody)
	})

	rc, _, err := c.Sessions.Subscribe(context.Background(), &SessionSubscribeRequest{SessionID: "s1", LastEventAt: 42})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer func() { _ = rc.Close() }()
	fr := NewSessionFrameReader(rc)
	n := 0
	for {
		_, err := fr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 2 {
		t.Errorf("frames = %d, want 2", n)
	}
}

func TestSessionsSubscribeHTTPErrorIsTyped(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"request_id":"r1","error":{"code":"AccessDenied","message":"not yours"}}`)
	})
	rc, _, err := c.Sessions.Subscribe(context.Background(), &SessionSubscribeRequest{SessionID: "s1"})
	if rc != nil {
		t.Error("ReadCloser must be nil on error")
	}
	var apiErr *ErrorResponse
	if !errors.As(err, &apiErr) || !strings.Contains(apiErr.Message, "not yours") {
		t.Fatalf("err = %v", err)
	}
}

func TestSessionsSubscribeOutlivesClientTimeoutAndHonorsCancel(t *testing.T) {
	release := make(chan struct{})
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, ": ping\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	t.Cleanup(func() { close(release) })
	// A short overall timeout would cut the stream if Subscribe reused it.
	c.client.Timeout = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	rc, _, err := c.Sessions.Subscribe(ctx, &SessionSubscribeRequest{SessionID: "s1"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer func() { _ = rc.Close() }()

	errCh := make(chan error, 1)
	go func() {
		_, err := NewSessionFrameReader(rc).Next()
		errCh <- err
	}()
	select {
	case err := <-errCh:
		t.Fatalf("stream ended after the client timeout: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-errCh:
		if err == nil || err == io.EOF {
			t.Fatalf("err after cancel = %v, want a cancellation error", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reader did not return after context cancel")
	}
}
