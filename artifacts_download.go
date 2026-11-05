package flashduty

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// Download streams the bytes of a presented file, using a token from
// ArtifactsService.ReadSign (the `t` query parameter of its download_url).
//
// The body is the file itself — any content type, any size — so it is returned
// as the live HTTP response body, never decoded or buffered. The caller owns
// the io.ReadCloser and MUST Close it. The request has no client-side overall
// timeout; bound it with ctx. resp.Header carries the file's Content-Type,
// Content-Length and Content-Disposition (with the file name).
//
// On a non-2xx status the JSON error envelope is surfaced as a typed error
// (*ErrorResponse, or *RateLimitError on 429) with a nil ReadCloser.
//
// API: GET /safari/artifact/stream (artifact-read-stream), mode=download.
func (s *ArtifactsService) Download(ctx context.Context, token string) (io.ReadCloser, *Response, error) {
	q := url.Values{"t": {token}, "mode": {"download"}}
	return s.client.openStream(ctx, http.MethodGet, "/safari/artifact/stream?"+q.Encode(), "*/*", nil)
}
