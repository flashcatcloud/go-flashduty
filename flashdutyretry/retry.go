// Package flashdutyretry provides a safe-by-default retrying http.RoundTripper
// for the Flashduty SDK, in the spirit of go-github's transport middleware.
//
// It plugs into the client as a composable transport layer:
//
//	client, err := flashduty.NewClient(appKey,
//		flashduty.WithTransport(flashdutyretry.New()),
//	)
//
// The Transport is pure net/http and deliberately does NOT import the parent
// flashduty package, so it can never introduce an import cycle.
//
// # Safe-by-default body replay
//
// A request is only ever retried when its body is replayable — that is, when
// req.Body is nil OR req.GetBody is non-nil. On each retry attempt the body is
// rebuilt from req.GetBody and attached to a per-attempt clone of the request,
// so the caller's original *http.Request is never mutated across attempts. If a
// retry would otherwise be warranted but the body cannot be replayed, the last
// response/error is returned as-is instead of risking a partial or empty body.
//
// The Flashduty SDK sets GetBody on every request it builds, so all SDK POST
// bodies are replayable; the guard exists so the middleware is still correct
// when handed arbitrary requests.
//
// # Retry policy
//
// Retries are attempted on HTTP 429, on any 5xx (status >= 500), and on
// transport errors (a non-nil error from the underlying RoundTripper). Other
// 4xx responses and all 2xx/3xx responses are returned immediately.
//
// Backoff is deterministic exponential: MinWait * 2^attempt, capped per-wait at
// MaxWait. A valid integer Retry-After header (delta-seconds) overrides the
// computed backoff (also capped at MaxWait). There is no randomized jitter —
// math/rand and time-based jitter are intentionally avoided.
//
// Context cancellation is honored while waiting between attempts: if the
// request's context is done, RoundTrip returns the context error.
package flashdutyretry

import (
	"io"
	"net/http"
	"strconv"
	"time"
)

// Default configuration applied for zero-valued Transport fields.
const (
	defaultMaxRetries = 3
	defaultMinWait    = 500 * time.Millisecond
	defaultMaxWait    = 30 * time.Second

	// drainLimit caps how many bytes we read while draining a discarded
	// response body so the underlying connection can be reused without
	// risking an unbounded read of a hostile/huge body.
	drainLimit = 4 << 20 // 4 MiB
)

// Transport is an http.RoundTripper that transparently retries idempotent and
// replayable requests on transient failures (HTTP 429, 5xx, and transport
// errors), using deterministic exponential backoff.
//
// All fields are optional: the zero value is usable and sensible defaults are
// applied inside RoundTrip, so both flashdutyretry.New() and
// &flashdutyretry.Transport{} produce a working transport.
type Transport struct {
	// Base is the underlying RoundTripper used to execute requests.
	// Defaults to http.DefaultTransport when nil.
	Base http.RoundTripper

	// MaxRetries is the maximum number of retry attempts after the first try.
	// Defaults to 3 when zero. A negative value disables retries entirely.
	MaxRetries int

	// MinWait is the base backoff duration (the wait before the first retry).
	// Defaults to 500ms when zero.
	MinWait time.Duration

	// MaxWait caps the duration of any single backoff wait.
	// Defaults to 30s when zero.
	MaxWait time.Duration
}

// Option configures a Transport in New.
type Option func(*Transport)

// WithBase sets the underlying RoundTripper. Nil falls back to
// http.DefaultTransport at request time.
func WithBase(base http.RoundTripper) Option {
	return func(t *Transport) { t.Base = base }
}

// WithMaxRetries sets the maximum number of retry attempts after the first try.
// A negative value disables retries.
func WithMaxRetries(n int) Option {
	return func(t *Transport) { t.MaxRetries = n }
}

// WithMinWait sets the base backoff duration.
func WithMinWait(d time.Duration) Option {
	return func(t *Transport) { t.MinWait = d }
}

// WithMaxWait sets the cap on any single backoff wait.
func WithMaxWait(d time.Duration) Option {
	return func(t *Transport) { t.MaxWait = d }
}

// New returns a *Transport configured by the given options. With no options it
// returns a Transport that uses http.DefaultTransport and the default retry
// budget (3 retries, 500ms..30s backoff).
func New(opts ...Option) *Transport {
	t := &Transport{}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// base returns the effective underlying RoundTripper.
func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// maxRetries returns the effective retry budget, applying the zero-value
// default while preserving an explicit negative (retries-disabled) setting.
func (t *Transport) maxRetries() int {
	if t.MaxRetries == 0 {
		return defaultMaxRetries
	}
	return t.MaxRetries
}

// minWait returns the effective base backoff.
func (t *Transport) minWait() time.Duration {
	if t.MinWait <= 0 {
		return defaultMinWait
	}
	return t.MinWait
}

// maxWait returns the effective per-wait cap.
func (t *Transport) maxWait() time.Duration {
	if t.MaxWait <= 0 {
		return defaultMaxWait
	}
	return t.MaxWait
}

// RoundTrip implements http.RoundTripper. It executes req and, on transient
// failures, retries up to MaxRetries times with deterministic exponential
// backoff — but only while the request body is replayable (see the package
// doc). It never mutates the caller's original *http.Request across attempts;
// each retry runs against a fresh clone with a rebuilt body.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	maxRetries := t.maxRetries()
	replayable := req.Body == nil || req.GetBody != nil

	var (
		resp *http.Response
		err  error
	)

	for attempt := 0; ; attempt++ {
		// Use a per-attempt clone so the original request is never mutated.
		// The first attempt could use req directly, but cloning uniformly
		// keeps the original pristine and the body handling consistent.
		ctx := req.Context()
		attemptReq := req.Clone(ctx)
		if attempt > 0 && req.GetBody != nil {
			body, gbErr := req.GetBody()
			if gbErr != nil {
				// Cannot rebuild the body — return what we have rather than
				// sending a request with a missing/empty body.
				return resp, err
			}
			attemptReq.Body = body
		}

		resp, err = t.base().RoundTrip(attemptReq)

		// Decide whether another attempt is warranted.
		if !shouldRetry(resp, err) {
			return resp, err
		}
		if attempt >= maxRetries {
			return resp, err
		}
		if !replayable {
			// A retry is warranted but the body cannot be replayed safely;
			// return the last response/error without retrying.
			return resp, err
		}

		// We are going to discard this response (if any) and try again, so
		// drain and close its body to allow connection reuse.
		drainAndClose(resp)

		wait := backoff(resp, attempt, t.minWait(), t.maxWait())

		// Wait for the backoff or context cancellation, whichever comes first.
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// shouldRetry reports whether the (resp, err) pair represents a transient
// failure that is eligible for a retry.
func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return false
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode >= 500
}

// backoff computes the wait before the next attempt. It prefers a valid integer
// Retry-After header (delta-seconds) when present, otherwise uses deterministic
// exponential backoff (min * 2^attempt). The result is always capped at max and
// never negative.
func backoff(resp *http.Response, attempt int, min, max time.Duration) time.Duration {
	if d, ok := retryAfter(resp); ok {
		return clamp(d, max)
	}

	wait := min
	for i := 0; i < attempt; i++ {
		wait *= 2
		if wait >= max {
			return max
		}
	}
	return clamp(wait, max)
}

// retryAfter parses an integer (delta-seconds) Retry-After header. It returns
// the duration and true only for a well-formed non-negative integer value;
// HTTP-date forms are deliberately not honored (the SDK's servers emit
// delta-seconds).
func retryAfter(resp *http.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0, false
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

// clamp bounds d to [0, max].
func clamp(d, max time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > max {
		return max
	}
	return d
}

// drainAndClose reads (up to drainLimit bytes) and closes a response body that
// is about to be discarded, so the underlying connection can be reused.
func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, drainLimit))
	_ = resp.Body.Close()
}
