package flashduty

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// noopLogger keeps test output clean.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient("KEY", WithBaseURL(srv.URL), WithLogger(noopLogger{}))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewRequestBuildsPostWithAppKeyAndJSON(t *testing.T) {
	c, _ := NewClient("KEY", WithBaseURL("https://api.flashcat.cloud"), WithLogger(noopLogger{}))
	req, err := c.newRequest(context.Background(), http.MethodPost, "/incident/list", map[string]any{"p": 1})
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("method = %s", req.Method)
	}
	if got := req.URL.Query().Get("app_key"); got != "KEY" {
		t.Fatalf("app_key = %q", got)
	}
	if req.URL.Path != "/incident/list" {
		t.Fatalf("path = %s", req.URL.Path)
	}
	if ct := req.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %s", ct)
	}
	body, _ := io.ReadAll(req.Body)
	if !strings.Contains(string(body), `"p":1`) {
		t.Fatalf("body = %s", body)
	}
}

func TestNewRequestAppliesHookAndHeaders(t *testing.T) {
	c, _ := NewClient("KEY",
		WithRequestHeaders(map[string][]string{"X-Static": {"s"}}),
		WithRequestHook(func(r *http.Request) { r.Header.Set("X-Hook", "h") }),
		WithLogger(noopLogger{}),
	)
	req, _ := c.newRequest(context.Background(), http.MethodPost, "/x", nil)
	if req.Header.Get("X-Static") != "s" || req.Header.Get("X-Hook") != "h" {
		t.Fatalf("headers not applied: %+v", req.Header)
	}
}

func TestDoDecodesDataAndPagination(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Flashcat-Request-Id", "RID1")
		_, _ = io.WriteString(w, `{"request_id":"RID1","data":{"total":2,"has_next_page":true,"search_after_ctx":"cur","items":[{"incident_id":"i1"}]}}`)
	})
	var out struct {
		Items []struct {
			IncidentID string `json:"incident_id"`
		} `json:"items"`
	}
	resp, err := c.do(context.Background(), "/incident/list", map[string]any{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if resp.RequestID != "RID1" || resp.Total != 2 || !resp.HasNextPage || resp.SearchAfterCtx != "cur" {
		t.Fatalf("response meta = %+v", resp)
	}
	if len(out.Items) != 1 || out.Items[0].IncidentID != "i1" {
		t.Fatalf("data not decoded: %+v", out)
	}
}

func TestDoMapsEnvelopeError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"request_id":"RID2","error":{"code":"incident_not_found","message":"nope"}}`)
	})
	_, err := c.do(context.Background(), "/incident/info", map[string]any{}, nil)
	var apiErr *ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Code != "incident_not_found" || apiErr.RequestID != "RID2" {
		t.Fatalf("expected mapped ErrorResponse, got %v", err)
	}
}

func TestDoMapsNon2xx(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = io.WriteString(w, `{"error":{"code":"internal","message":"boom"}}`)
	})
	_, err := c.do(context.Background(), "/x", map[string]any{}, nil)
	var apiErr *ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response.StatusCode != 500 {
		t.Fatalf("expected 500 ErrorResponse, got %v", err)
	}
}

func TestDoEmptyBodySuccess(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"request_id":"RID3"}`)
	})
	resp, err := c.do(context.Background(), "/incident/ack", map[string]any{}, nil)
	if err != nil || resp.RequestID != "RID3" {
		t.Fatalf("empty-data success failed: err=%v resp=%+v", err, resp)
	}
}

func TestDoReturnsRateLimitErrorOn429(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(429)
		_, _ = io.WriteString(w, `{"error":{"code":"rate_limited","message":"slow down"}}`)
	})
	resp, err := c.do(context.Background(), "/incident/list", map[string]any{}, nil)
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rl.RetryAfter != 30*time.Second {
		t.Fatalf("RetryAfter = %s, want 30s", rl.RetryAfter)
	}
	var generic *ErrorResponse
	if !errors.As(err, &generic) || generic.Code != "rate_limited" {
		t.Fatalf("RateLimitError must unwrap to *ErrorResponse")
	}
	if resp.RateLimit.RetryAfter != 30*time.Second || resp.RateLimit.Remaining != 0 {
		t.Fatalf("Response.RateLimit not populated: %+v", resp.RateLimit)
	}
}
