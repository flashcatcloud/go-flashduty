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

// ndjsonBody is a realistic export stream: a session_meta envelope first, then a
// run of event lines. A trailing newline (as the real endpoint emits) must not
// produce a spurious empty token.
const ndjsonBody = `{"type":"session_meta","session_id":"sess_abc","account_id":2451002751131,"app_name":"ai-sre","model":"deepseek-v4-pro"}
{"type":"user_message","seq":1,"ts":"2026-06-02T02:39:31.241Z","content":"hi"}
{"type":"tool_call","seq":2,"ts":"2026-06-02T02:39:32.000Z","name":"shell","status":"ok","output_bytes":42}
{"type":"final_answer","seq":3,"ts":"2026-06-02T02:39:53.457Z","content":"done"}
`

func TestExportStreamsNDJSONLineByLine(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Endpoint, app_key injection, and the NDJSON Accept override.
		if r.URL.Path != "/safari/session/export" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("app_key"); got != "KEY" {
			t.Errorf("app_key = %q", got)
		}
		if acc := r.Header.Get("Accept"); acc != "application/x-ndjson" {
			t.Errorf("Accept = %q, want application/x-ndjson", acc)
		}
		// Request body carries the typed request.
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"session_id":"sess_abc"`) {
			t.Errorf("request body = %s", body)
		}
		w.Header().Set("Flashcat-Request-Id", "RIDX")
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, ndjsonBody)
	})

	rc, resp, err := c.Sessions.Export(context.Background(), &SessionExportRequest{SessionID: "sess_abc"})
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}
	if rc == nil {
		t.Fatal("Export returned nil ReadCloser")
	}
	defer func() { _ = rc.Close() }()
	if resp == nil || resp.RequestID != "RIDX" {
		t.Fatalf("response meta = %+v", resp)
	}

	// Consume line-by-line via the helper scanner; assert each line decodes and
	// that session_meta is first.
	sc := NewExportScanner(rc)
	var types []string
	var firstSessionID string
	for sc.Scan() {
		line, derr := DecodeExportLine(sc.Bytes())
		if derr != nil {
			t.Fatalf("decode line %q: %v", sc.Bytes(), derr)
		}
		types = append(types, line.Type)
		if line.Type == "session_meta" {
			firstSessionID = line.SessionID
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	want := []string{"session_meta", "user_message", "tool_call", "final_answer"}
	if len(types) != len(want) {
		t.Fatalf("got %d lines %v, want %d %v", len(types), types, len(want), want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("line %d type = %q, want %q (all: %v)", i, types[i], want[i], types)
		}
	}
	if types[0] != "session_meta" {
		t.Fatalf("first line must be session_meta, got %q", types[0])
	}
	if firstSessionID != "sess_abc" {
		t.Fatalf("session_meta.session_id = %q, want sess_abc", firstSessionID)
	}
}

func TestExportMapsErrorEnvelopeOnNon2xx(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"request_id":"RIDE","error":{"code":"access_denied","message":"no"}}`)
	})
	rc, resp, err := c.Sessions.Export(context.Background(), &SessionExportRequest{SessionID: "sess_x"})
	if rc != nil {
		t.Fatal("ReadCloser must be nil on error")
	}
	var apiErr *ErrorResponse
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *ErrorResponse, got %T: %v", err, err)
	}
	if apiErr.Code != "access_denied" || apiErr.RequestID != "RIDE" {
		t.Fatalf("error not mapped: %+v", apiErr)
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("response status = %+v", resp)
	}
}

func TestExportReturnsRateLimitErrorOn429(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"code":"rate_limited","message":"slow down"}}`)
	})
	rc, _, err := c.Sessions.Export(context.Background(), &SessionExportRequest{SessionID: "sess_x"})
	if rc != nil {
		t.Fatal("ReadCloser must be nil on 429")
	}
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rl.RetryAfter != 30*time.Second {
		t.Fatalf("RetryAfter = %s, want 30s", rl.RetryAfter)
	}
}
