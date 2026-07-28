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

// TestOptionalObjectRequestFieldOmitsWhenUnset guards against a codegen
// regression: encoding/json's `,omitempty` never drops a bare struct value
// (only false/0/""/nil-slice/nil-map/nil-pointer count as "empty"), so an
// unset optional object request field used to always be sent as `{}`. For
// CreateSilenceRuleRequest.TimeFilter this made a recurring-only silence rule
// (TimeFilters set, TimeFilter unset) impossible: the server's binding
// validates StartTime/EndTime with `gt=0` whenever "time_filter" is present
// in the payload at all. The generator now emits `,omitzero` for optional
// struct-typed request fields, which correctly drops the zero value.
func TestOptionalObjectRequestFieldOmitsWhenUnset(t *testing.T) {
	c, _ := NewClient("KEY", WithBaseURL("https://api.flashcat.cloud"), WithLogger(noopLogger{}))

	req, err := c.newRequest(context.Background(), http.MethodPost, "/silence-rule/create", &CreateSilenceRuleRequest{
		RuleName: "recurring only",
		TimeFilters: []CreateSilenceRuleRequestTimeFiltersItem{
			{Start: "09:00", End: "18:00"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(req.Body)
	if strings.Contains(string(body), `"time_filter"`) {
		t.Fatalf("unset TimeFilter must be omitted from the wire, got body = %s", body)
	}
	if !strings.Contains(string(body), `"time_filters"`) {
		t.Fatalf("TimeFilters must be present, got body = %s", body)
	}

	req, err = c.newRequest(context.Background(), http.MethodPost, "/silence-rule/create", &CreateSilenceRuleRequest{
		RuleName:   "one-off only",
		TimeFilter: CreateSilenceRuleRequestTimeFilter{StartTime: 1000, EndTime: 2000},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(req.Body)
	if !strings.Contains(string(body), `"time_filter"`) {
		t.Fatalf("set TimeFilter must be present on the wire, got body = %s", body)
	}
}

// TestResetPostMortemContentSendsZeroExpectedRevision guards a codegen
// contract: expected_revision is a required field where 0 is a valid value
// (first write to a never-saved document, per the spec's minimum: 0), so it
// must reach the wire even when zero. The spec models it as
// type: ["integer", "null"], which the generator rewrites to a pointer — a
// nil pointer means "unset" and is omitted, while Int64(0) is sent.
func TestResetPostMortemContentSendsZeroExpectedRevision(t *testing.T) {
	c, _ := NewClient("KEY", WithBaseURL("https://api.flashcat.cloud"), WithLogger(noopLogger{}))

	req, err := c.newRequest(context.Background(), http.MethodPost, "/incident/post-mortem/content/reset", &ResetPostMortemContentRequest{
		PostMortemID:     "pm_x",
		Markdown:         "## impact",
		ExpectedRevision: Int64(0),
		IdempotencyKey:   "key-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(req.Body)
	if !strings.Contains(string(body), `"expected_revision":0`) {
		t.Fatalf("ExpectedRevision=Int64(0) must be sent on the wire, got body = %s", body)
	}

	req, err = c.newRequest(context.Background(), http.MethodPost, "/incident/post-mortem/content/reset", &ResetPostMortemContentRequest{
		PostMortemID:   "pm_x",
		Markdown:       "## impact",
		IdempotencyKey: "key-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(req.Body)
	if strings.Contains(string(body), `"expected_revision"`) {
		t.Fatalf("nil ExpectedRevision must be omitted from the wire, got body = %s", body)
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

func TestDoTreatsOKCodeAsSuccess(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Some success envelopes carry error.code "OK" rather than a null error.
		_, _ = io.WriteString(w, `{"request_id":"RID","error":{"code":"OK","message":""},"data":{"items":[{"incident_id":"i1"}]}}`)
	})
	var out struct {
		Items []struct {
			IncidentID string `json:"incident_id"`
		} `json:"items"`
	}
	resp, err := c.do(context.Background(), "/incident/list", map[string]any{}, &out)
	if err != nil {
		t.Fatalf("OK code must be treated as success, got %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].IncidentID != "i1" || resp.RequestID != "RID" {
		t.Fatalf("data not decoded on OK envelope: out=%+v resp=%+v", out, resp)
	}
}

// TestDoReportsIntermediaryOnNon2xxNonJSONBody guards a prod incident: an ALB
// timed out a long-running request and returned an HTML 504 page with no
// Flashcat-Request-Id header. The error must name the status, say the
// response came from an intermediary rather than the Flashduty API, note the
// missing request id, hint at retry/splitting the batch, and echo a body
// snippet — not the old "malformed response" wording, which reads like an
// API bug.
func TestDoReportsIntermediaryOnNon2xxNonJSONBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = io.WriteString(w, "<html><body>504 Gateway Time-out</body></html>")
	})
	_, err := c.do(context.Background(), "/incident/list", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	msg := err.Error()
	for _, want := range []string{"HTTP 504", "intermediary", "no request id", "gateway/proxy timeout", "504 Gateway Time-out"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error message = %q, want it to contain %q", msg, want)
		}
	}
	if strings.Contains(msg, "malformed response") {
		t.Fatalf("error message = %q, must not use the old malformed-response wording", msg)
	}
}

// TestDoReportsIntermediaryWithRequestID covers the case where the
// intermediary (or an upstream hop before it) did forward a request id.
func TestDoReportsIntermediaryWithRequestID(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Flashcat-Request-Id", "RID-504")
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = io.WriteString(w, "<html>timeout</html>")
	})
	_, err := c.do(context.Background(), "/incident/list", map[string]any{}, nil)
	if err == nil || !strings.Contains(err.Error(), "request id RID-504") {
		t.Fatalf("expected error to include the forwarded request id, got %v", err)
	}
}

// TestDoMapsNon2xxJSONEnvelopeUnaffectedByGatewaySplit confirms a real 504
// bearing a normal JSON error envelope is completely unaffected by the new
// non-JSON/intermediary branch: it still maps to *ErrorResponse as before.
func TestDoMapsNon2xxJSONEnvelopeUnaffectedByGatewaySplit(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Flashcat-Request-Id", "RID4")
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = io.WriteString(w, `{"request_id":"RID4","error":{"code":"timeout","message":"upstream took too long"}}`)
	})
	_, err := c.do(context.Background(), "/incident/list", map[string]any{}, nil)
	var apiErr *ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Code != "timeout" || apiErr.Response.StatusCode != http.StatusGatewayTimeout || apiErr.RequestID != "RID4" {
		t.Fatalf("expected mapped ErrorResponse for JSON envelope, got %v", err)
	}
}

// TestDoExposesNonJSONBodyAsRaw also guards the 2xx branch against the
// non-2xx/intermediary-error split above: a non-JSON body on success must
// keep returning Response.Raw with no error, never the new gateway message.
func TestDoExposesNonJSONBodyAsRaw(t *testing.T) {
	const csv = "id,title\n1,boom\n2,bam\n"
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, csv)
	})
	resp, err := c.do(context.Background(), "/insight/incident/export", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("non-JSON success body must not error, got %v", err)
	}
	if string(resp.Raw) != csv {
		t.Fatalf("Response.Raw = %q, want the CSV body", resp.Raw)
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
