package flashduty

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Export streams a session's full event transcript as newline-delimited JSON
// (application/x-ndjson). The first line is always a session_meta envelope;
// subsequent lines are session events. When req.IncludeSubagents is true, each
// subagent_dispatch line is followed by the child session's own stream.
//
// Unlike the generated typed endpoints, the success body is NOT a JSON envelope:
// it is a potentially large line-delimited stream meant to be written straight to
// a file. The returned io.ReadCloser is the live HTTP response body — the caller
// owns it and MUST Close it (a deferred close is correct). Parse it line-by-line
// (see NewExportScanner / DecodeExportLine); do not buffer the whole transcript
// into memory.
//
// On any non-2xx status the body is the usual JSON error envelope: Export reads
// and closes it and returns a typed error (*ErrorResponse, or *RateLimitError on
// 429) with a nil ReadCloser, matching the generated endpoints.
//
// API: POST /safari/session/export (session-read-export).
func (s *SessionsService) Export(ctx context.Context, req *SessionExportRequest) (io.ReadCloser, *Response, error) {
	httpReq, err := s.client.newRequest(ctx, http.MethodPost, "/safari/session/export", req)
	if err != nil {
		return nil, nil, err
	}
	// The success body is a stream, not a JSON envelope; ask for NDJSON.
	httpReq.Header.Set("Accept", "application/x-ndjson")

	httpResp, err := s.client.client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("flashduty: request to %s failed: %v", sanitizeURL(httpReq.URL), sanitizeError(err))
	}

	resp := &Response{Response: httpResp, RequestID: httpResp.Header.Get("Flashcat-Request-Id")}
	resp.RateLimit = parseRateLimit(httpResp.Header)

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		// Error responses are JSON envelopes, even on a streaming endpoint. Drain
		// (bounded) and close the body, then surface a typed error.
		defer func() { _ = httpResp.Body.Close() }()
		raw, _ := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseBodySize))
		var env envelope
		_ = json.Unmarshal(raw, &env)
		if env.RequestID != "" {
			resp.RequestID = env.RequestID
		}
		apiErr := &ErrorResponse{
			Response:  httpResp,
			Code:      env.Error.codeOr(""),
			Message:   env.Error.errMessageOr(string(raw)),
			RequestID: resp.RequestID,
		}
		return nil, resp, asAPIError(apiErr, resp.RateLimit)
	}

	// 2xx: hand the live stream to the caller, who owns Close.
	return httpResp.Body, resp, nil
}

// NewExportScanner wraps an export stream in a bufio.Scanner configured to read
// one NDJSON line per Scan, with a buffer large enough for the wide event lines
// (tool output, llm calls) that the transcript can contain. Each token is one raw
// JSON line; decode it with DecodeExportLine or json.Unmarshal into ExportLine.
//
//	rc, _, err := client.Sessions.Export(ctx, req)
//	if err != nil { return err }
//	defer rc.Close()
//	sc := flashduty.NewExportScanner(rc)
//	for sc.Scan() {
//	    line, err := flashduty.DecodeExportLine(sc.Bytes())
//	    // ... handle line, or write sc.Bytes() straight to a file ...
//	}
//	return sc.Err()
func NewExportScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	// A single transcript line (e.g. a large tool_call output) can exceed the
	// 64KB default token limit; allow up to maxResponseBodySize per line.
	sc.Buffer(make([]byte, 0, 64*1024), maxResponseBodySize)
	return sc
}

// DecodeExportLine unmarshals one raw NDJSON export line into an ExportLine. Use
// the Type field to discriminate (session_meta, user_message, llm_call,
// tool_call, subagent_dispatch, final_answer, agent_text, error).
func DecodeExportLine(line []byte) (*ExportLine, error) {
	var l ExportLine
	if err := json.Unmarshal(line, &l); err != nil {
		return nil, fmt.Errorf("flashduty: decoding export line: %w", err)
	}
	return &l, nil
}
