package retry

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// fakeRT is a scripted http.RoundTripper. For each call it returns the next
// (response, error) pair from its scripts, counts calls, and records the body
// bytes it received on every attempt.
type fakeRT struct {
	statuses []int   // per-attempt status codes (0 ⇒ use err instead of a response)
	errs     []error // per-attempt errors (nil ⇒ use the status code)
	header   http.Header

	calls     int
	gotBodies []string
}

func (f *fakeRT) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := f.calls
	f.calls++

	// Record the body bytes seen on this attempt.
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		f.gotBodies = append(f.gotBodies, string(b))
	} else {
		f.gotBodies = append(f.gotBodies, "")
	}

	if idx < len(f.errs) && f.errs[idx] != nil {
		return nil, f.errs[idx]
	}

	status := http.StatusOK
	if idx < len(f.statuses) && f.statuses[idx] != 0 {
		status = f.statuses[idx]
	}

	h := http.Header{}
	if f.header != nil {
		h = f.header.Clone()
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader("body")),
	}, nil
}

func fastTransport(base http.RoundTripper, max int) *Transport {
	return New(
		WithBase(base),
		WithMaxRetries(max),
		WithMinWait(time.Millisecond),
		WithMaxWait(5*time.Millisecond),
	)
}

func newGetReq(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://example.test/x", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	return req
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	base := &fakeRT{statuses: []int{http.StatusTooManyRequests, http.StatusOK}}
	tr := fastTransport(base, 3)

	resp, err := tr.RoundTrip(newGetReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if base.calls != 2 {
		t.Fatalf("calls = %d, want 2", base.calls)
	}
}

func TestRetryOn500GivesUp(t *testing.T) {
	const maxRetries = 2
	base := &fakeRT{statuses: []int{
		http.StatusInternalServerError,
		http.StatusInternalServerError,
		http.StatusInternalServerError,
	}}
	tr := fastTransport(base, maxRetries)

	resp, err := tr.RoundTrip(newGetReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if base.calls != maxRetries+1 {
		t.Fatalf("calls = %d, want %d", base.calls, maxRetries+1)
	}
}

func TestNoRetryOn400(t *testing.T) {
	base := &fakeRT{statuses: []int{http.StatusBadRequest}}
	tr := fastTransport(base, 3)

	resp, err := tr.RoundTrip(newGetReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if base.calls != 1 {
		t.Fatalf("calls = %d, want 1", base.calls)
	}
}

func TestRetryOnTransportErrorThenSuccess(t *testing.T) {
	base := &fakeRT{
		statuses: []int{0, http.StatusOK},
		errs:     []error{errors.New("dial tcp: connection refused"), nil},
	}
	tr := fastTransport(base, 3)

	resp, err := tr.RoundTrip(newGetReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if base.calls != 2 {
		t.Fatalf("calls = %d, want 2", base.calls)
	}
}

func TestHonorsRetryAfterHeader(t *testing.T) {
	base := &fakeRT{
		statuses: []int{http.StatusTooManyRequests, http.StatusOK},
		header:   http.Header{"Retry-After": []string{"0"}},
	}
	tr := fastTransport(base, 3)

	start := time.Now()
	resp, err := tr.RoundTrip(newGetReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if base.calls != 2 {
		t.Fatalf("calls = %d, want 2 (should have retried)", base.calls)
	}
	// Retry-After: 0 ⇒ effectively immediate; sanity-check it didn't block.
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("retry took %v, expected near-immediate with Retry-After: 0", elapsed)
	}
}

func TestReplaysPOSTBodyAcrossAttempts(t *testing.T) {
	const payload = "payload"
	base := &fakeRT{statuses: []int{
		http.StatusServiceUnavailable,
		http.StatusServiceUnavailable,
		http.StatusOK,
	}}
	tr := fastTransport(base, 3)

	// http.NewRequest sets GetBody automatically for a *strings.Reader body.
	req, err := http.NewRequest(http.MethodPost, "http://example.test/x", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if req.GetBody == nil {
		t.Fatal("expected GetBody to be set by http.NewRequest")
	}

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if base.calls != 3 {
		t.Fatalf("calls = %d, want 3", base.calls)
	}
	if len(base.gotBodies) != 3 {
		t.Fatalf("recorded %d bodies, want 3", len(base.gotBodies))
	}
	for i, got := range base.gotBodies {
		if got != payload {
			t.Fatalf("attempt %d body = %q, want %q (body not replayed)", i, got, payload)
		}
	}
}

func TestRespectsContextCancellation(t *testing.T) {
	// Always 500 so a retry (and thus a backoff wait) is always warranted.
	base := &fakeRT{statuses: []int{
		http.StatusInternalServerError,
		http.StatusInternalServerError,
	}}
	// Use a long MinWait so the wait is dominated by the backoff select,
	// giving the cancellation a clear window to win.
	tr := New(
		WithBase(base),
		WithMaxRetries(3),
		WithMinWait(time.Hour),
		WithMaxWait(time.Hour),
	)

	ctx, cancel := context.WithCancel(context.Background())
	req := newGetReq(t).WithContext(ctx)

	// Cancel shortly after RoundTrip enters its backoff wait.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := tr.RoundTrip(req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestNoRetryWhenBodyNotReplayable(t *testing.T) {
	base := &fakeRT{statuses: []int{
		http.StatusInternalServerError,
		http.StatusOK, // should never be reached
	}}
	tr := fastTransport(base, 3)

	req, err := http.NewRequest(http.MethodPost, "http://example.test/x", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	// Make the body non-replayable.
	req.GetBody = nil

	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if base.calls != 1 {
		t.Fatalf("calls = %d, want 1 (must not retry non-replayable body)", base.calls)
	}
}
