package flashduty

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// uploadFile streams a multipart/form-data POST: the file rides the "file"
// form part (both upload endpoints use that part name), fields carry the
// remaining form values, and query carries extra query parameters (app_key is
// added here like on every other request). The body streams through an io.Pipe
// so large archives never buffer in memory. Response handling goes through
// processResponse, so envelope errors and typed data behave exactly as on the
// generated JSON path; pass out=nil when the endpoint returns no data payload.
func (c *Client) uploadFile(ctx context.Context, path string, query url.Values, fields map[string]string, filename string, file io.Reader, out any) (*Response, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		// WriteField before CreateFormFile: multipart parts are positional,
		// and some servers stop reading once they have seen every part they
		// know. Sort keys so the wire order is deterministic.
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var err error
		for _, k := range keys {
			if err = mw.WriteField(k, fields[k]); err != nil {
				break
			}
		}
		if err == nil {
			var part io.Writer
			part, err = mw.CreateFormFile("file", filename)
			if err == nil {
				_, err = io.Copy(part, file)
			}
		}
		if cerr := mw.Close(); err == nil {
			err = cerr
		}
		// CloseWithError surfaces a writer-side failure to the HTTP client as
		// a body-read error on the request.
		_ = pw.CloseWithError(err)
	}()

	rel, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("flashduty: invalid path %q: %w", path, err)
	}
	u := c.BaseURL.ResolveReference(rel)
	q := u.Query()
	q.Set("app_key", c.appKey)
	for k, vs := range query {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), pr)
	if err != nil {
		return nil, fmt.Errorf("flashduty: building upload request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
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
		"method", http.MethodPost,
		"url", sanitizeURL(u),
		"body", "<multipart "+filename+">",
	)

	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flashduty: request to %s failed: %v", sanitizeURL(req.URL), sanitizeError(err))
	}
	return c.processResponse(httpResp, out)
}
