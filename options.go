package flashduty

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Option configures a Client during NewClient.
type Option func(*Client)

// WithBaseURL overrides the API base URL (default https://api.flashcat.cloud).
func WithBaseURL(raw string) Option {
	parsed, err := url.Parse(raw)
	return func(c *Client) {
		if err != nil || parsed == nil || parsed.Host == "" {
			c.optionErr = fmt.Errorf("flashduty: invalid base URL %q: %w", raw, err)
			return
		}
		c.BaseURL = parsed
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.client.Timeout = d }
}

// WithUserAgent sets the User-Agent header sent on every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.UserAgent = ua }
}

// WithHTTPClient supplies a custom *http.Client. Nil is ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.client = hc
		}
	}
}

// WithTransport sets a custom http.RoundTripper on the underlying client — the
// idiomatic seam for retry, caching, tracing, or rate-limit middleware (see the
// retry subpackage). Nil is ignored.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *Client) {
		if rt != nil {
			c.client.Transport = rt
		}
	}
}

// WithLogger sets a custom Logger. Nil is ignored.
func WithLogger(l Logger) Option {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithRequestHeaders sets static headers added to every request, applied after
// the SDK's own headers (Content-Type, Accept, User-Agent).
func WithRequestHeaders(h http.Header) Option {
	return func(c *Client) { c.requestHeaders = h }
}

// WithRequestHook registers a callback invoked on every outgoing request before
// it is sent — use it to inject per-request headers (e.g. W3C traceparent).
func WithRequestHook(hook func(*http.Request)) Option {
	return func(c *Client) { c.requestHook = hook }
}
