package flashduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL      = "https://api.flashcat.cloud"
	defaultUserAgent    = "go-flashduty"
	maxResponseBodySize = 10 << 20 // 10MB guard against OOM
)

// Client is a Flashduty Open API client. Construct it with NewClient and use the
// service fields (added by the generator) to call endpoints.
type Client struct {
	client    *http.Client
	BaseURL   *url.URL
	UserAgent string

	appKey         string
	logger         Logger
	requestHeaders http.Header
	requestHook    func(*http.Request)
	optionErr      error

	common service // shared backref reused by every service (one allocation)

	// genServices (generated in services_gen.go) embeds the typed service
	// handles — c.Incidents, c.Alerts, … — wired by initServices.
	genServices
}

// NewClient returns a Flashduty client authenticated with the given app key.
func NewClient(appKey string, opts ...Option) (*Client, error) {
	if appKey == "" {
		return nil, fmt.Errorf("flashduty: app key is required")
	}
	base, _ := url.Parse(defaultBaseURL)
	c := &Client{
		client:    &http.Client{Timeout: 30 * time.Second},
		BaseURL:   base,
		UserAgent: defaultUserAgent,
		appKey:    appKey,
		logger:    defaultLogger,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.optionErr != nil {
		return nil, c.optionErr
	}
	c.initServices()
	return c, nil
}

// Response wraps http.Response and surfaces Flashduty envelope metadata: the
// request id (for support), pagination fields when the endpoint returns them,
// and best-effort rate-limit signals.
type Response struct {
	*http.Response

	RequestID      string
	Total          int
	HasNextPage    bool
	SearchAfterCtx string
	RateLimit      RateLimit

	// Raw holds the response body for endpoints that return a non-JSON payload
	// on success (e.g. CSV/file downloads from the *export endpoints). It is nil
	// for normal JSON responses; the typed return value is then the zero value.
	Raw []byte
}

// envelope is the universal {request_id, error, data} response wrapper.
type envelope struct {
	RequestID string          `json:"request_id"`
	Error     *DutyError      `json:"error"`
	Data      json.RawMessage `json:"data"`
}

// pageMeta mirrors the pagination fields the backend nests inside data.
type pageMeta struct {
	Total          int    `json:"total"`
	HasNextPage    bool   `json:"has_next_page"`
	SearchAfterCtx string `json:"search_after_ctx"`
}

// newRequest builds an HTTP request to path, injecting the app_key query
// parameter and JSON-encoding body when non-nil. Most Flashduty endpoints are
// POST actions; a handful are GET with query parameters.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	return c.newRequestWithAppKey(ctx, method, path, body, true)
}

func (c *Client) newRequestWithAppKey(ctx context.Context, method, path string, body any, withAppKey bool) (*http.Request, error) {
	rel, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("flashduty: invalid path %q: %w", path, err)
	}
	u := c.BaseURL.ResolveReference(rel)
	if withAppKey {
		q := u.Query()
		q.Set("app_key", c.appKey)
		u.RawQuery = q.Encode()
	}

	var buf io.Reader
	var rawBody []byte
	if body != nil {
		rawBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("flashduty: encoding request body: %w", err)
		}
		buf = bytes.NewReader(rawBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("flashduty: building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	for k, vs := range c.requestHeaders {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	if c.requestHook != nil {
		c.requestHook(req)
	}

	c.logger.Info("flashduty request",
		"method", method,
		"url", sanitizeURL(u),
		"body", truncateBody(sanitizeBody(string(rawBody))),
	)
	return req, nil
}

// do performs a POST to path with a JSON body. Most generated endpoints are
// POST actions and call this.
func (c *Client) do(ctx context.Context, path string, body, out any) (*Response, error) {
	return c.doMethod(ctx, http.MethodPost, path, body, out)
}

// doGet performs a GET to path, encoding opt's `url`-tagged fields as query
// parameters. The handful of GET endpoints call this.
func (c *Client) doGet(ctx context.Context, path string, opt, out any) (*Response, error) {
	full, err := addQueryParams(path, opt)
	if err != nil {
		return nil, fmt.Errorf("flashduty: encoding query for %s: %w", path, err)
	}
	return c.doMethod(ctx, http.MethodGet, full, nil, out)
}

// doMethod performs the request, unwraps the envelope, decodes data into out
// (when non-nil), and returns a Response. A non-nil envelope error or a non-2xx
// status yields an *ErrorResponse (or *RateLimitError on 429).
func (c *Client) doMethod(ctx context.Context, method, path string, body, out any) (*Response, error) {
	return c.doMethodWithAppKey(ctx, method, path, body, out, true, nil)
}

func (c *Client) doMethodWithoutAppKey(ctx context.Context, method, path string, body, out any, configure func(*http.Request)) (*Response, error) {
	return c.doMethodWithAppKey(ctx, method, path, body, out, false, configure)
}

func (c *Client) doMethodWithAppKey(ctx context.Context, method, path string, body, out any, withAppKey bool, configure func(*http.Request)) (*Response, error) {
	req, err := c.newRequestWithAppKey(ctx, method, path, body, withAppKey)
	if err != nil {
		return nil, err
	}
	if configure != nil {
		configure(req)
	}
	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flashduty: request to %s failed: %v", sanitizeURL(req.URL), sanitizeError(err))
	}
	defer func() { _ = httpResp.Body.Close() }()
	return c.processResponse(httpResp, out)
}

// processResponse drains and interprets an HTTP response: it unwraps the
// {request_id, error, data} envelope, decodes data into out (when non-nil),
// and surfaces non-JSON success bodies on Response.Raw. Hand-written methods
// that build their own request (e.g. multipart upload) reuse it so envelope
// handling lives in exactly one place. The caller that issued the request
// owns closing httpResp.Body; processResponse only reads it.
func (c *Client) processResponse(httpResp *http.Response, out any) (*Response, error) {
	resp := &Response{Response: httpResp, RequestID: httpResp.Header.Get("Flashcat-Request-Id")}
	resp.RateLimit = parseRateLimit(httpResp.Header)

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBodySize))
	if err != nil {
		return resp, fmt.Errorf("flashduty: reading response body: %w", err)
	}
	c.logger.Info("flashduty response",
		"status", httpResp.StatusCode,
		"body", truncateBody(sanitizeBody(string(raw))),
	)

	var env envelope
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			// Some success endpoints return a non-JSON body (e.g. CSV/file
			// downloads from the *export endpoints, despite the spec declaring
			// JSON). Surface those bytes on Response.Raw instead of erroring.
			if httpResp.StatusCode/100 == 2 {
				resp.Raw = raw
				return resp, nil
			}
			// A non-2xx status with a body that isn't valid envelope JSON never
			// came from the Flashduty API itself — it always answers in JSON.
			// It's an intermediary (gateway/load balancer/proxy) cutting the
			// request off, most often on its own timeout, and returning an
			// HTML/plaintext error page instead. Report that distinctly so it
			// isn't mistaken for a Flashduty API contract violation.
			requestIDNote := "no request id"
			if resp.RequestID != "" {
				requestIDNote = "request id " + resp.RequestID
			}
			return resp, fmt.Errorf("flashduty: HTTP %d returned by an intermediary, not the Flashduty API (non-JSON body, %s) — the request likely exceeded a gateway/proxy timeout; retry, or split long-running requests into smaller batches",
				httpResp.StatusCode, requestIDNote)
		}
	}
	if env.RequestID != "" {
		resp.RequestID = env.RequestID
	}

	if env.Error != nil && isFailureCode(env.Error.Code) {
		apiErr := &ErrorResponse{Response: httpResp, Code: env.Error.Code, Message: env.Error.Message, RequestID: resp.RequestID}
		return resp, asAPIError(apiErr, resp.RateLimit)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		apiErr := &ErrorResponse{Response: httpResp, Code: env.Error.codeOr(""), Message: env.Error.errMessageOr(string(raw)), RequestID: resp.RequestID}
		return resp, asAPIError(apiErr, resp.RateLimit)
	}

	if len(env.Data) > 0 {
		var meta pageMeta
		if json.Unmarshal(env.Data, &meta) == nil {
			resp.Total, resp.HasNextPage, resp.SearchAfterCtx = meta.Total, meta.HasNextPage, meta.SearchAfterCtx
		}
		if out != nil {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return resp, fmt.Errorf("flashduty: decoding data into %T (request_id %s): %w", out, resp.RequestID, err)
			}
		}
	}
	return resp, nil
}

// isFailureCode reports whether an envelope error.code denotes a real failure.
// Empty, "0", and "OK" (the ErrorCodeOK success sentinel) are not failures.
func isFailureCode(code string) bool {
	switch {
	case code == "", code == "0":
		return false
	case strings.EqualFold(code, string(ErrorCodeOK)):
		return false
	default:
		return true
	}
}

// asAPIError promotes a 429 ErrorResponse to a *RateLimitError so callers can
// type-switch on rate limiting; other statuses pass through unchanged.
func asAPIError(e *ErrorResponse, rl RateLimit) error {
	if e.Response != nil && e.Response.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{ErrorResponse: e, RetryAfter: rl.RetryAfter}
	}
	return e
}
