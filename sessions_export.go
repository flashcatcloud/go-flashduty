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

// ExportLine is one decoded line of a session export stream. The Type field
// discriminates the event kind (session_meta, user_message, llm_call, tool_call,
// subagent_dispatch, final_answer, agent_text, error); the remaining fields are
// populated per kind. It is hand-maintained here, alongside the hand-written
// streaming Export endpoint, because the typed generator only models JSON-envelope
// responses and excludes the NDJSON export stream (so its line schema is pruned as
// unreferenced).
type ExportLine struct {
	// Account id (on session_meta).
	AccountID int64 `json:"account_id" toon:"account_id"`
	// Dispatched subagent name (on subagent_dispatch).
	AgentName string `json:"agent_name" toon:"agent_name"`
	// Agent app (on session_meta).
	AppName string `json:"app_name" toon:"app_name"`
	// Child session id created by the dispatch (on subagent_dispatch).
	ChildSessionID string `json:"child_session_id" toon:"child_session_id"`
	// Text content of the line (messages, answers, errors).
	Content string `json:"content" toon:"content"`
	// Call duration in milliseconds.
	DurationMs int64 `json:"duration_ms" toon:"duration_ms"`
	// RFC3339 end timestamp; stamped on llm_call/tool_call/session_meta.
	EndedAt string `json:"ended_at" toon:"ended_at"`
	// Error detail when a call failed.
	Error string `json:"error" toon:"error"`
	// Tool call input arguments (on tool_call).
	Input map[string]any `json:"input" toon:"input"`
	// Byte size of the tool input.
	InputBytes int64 `json:"input_bytes" toon:"input_bytes"`
	// Chat model provider key; on session_meta and llm_call.
	Model string `json:"model" toon:"model"`
	// Tool name (on tool_call).
	Name string `json:"name" toon:"name"`
	// Tool call output (on tool_call response side).
	Output string `json:"output" toon:"output"`
	// Byte size of the tool output.
	OutputBytes int64 `json:"output_bytes" toon:"output_bytes"`
	// Parent session id for child sessions (on session_meta).
	ParentSessionID string `json:"parent_session_id" toon:"parent_session_id"`
	// 1-based monotonic sequence within the session (absent on session_meta).
	Seq int64 `json:"seq" toon:"seq"`
	// Session id (on session_meta).
	SessionID string `json:"session_id" toon:"session_id"`
	// RFC3339 start timestamp (session_meta uses session.created_at).
	StartedAt string `json:"started_at" toon:"started_at"`
	// Tool result status, e.g. ok or error.
	Status string `json:"status" toon:"status"`
	// RFC3339 timestamp of the event.
	TS string `json:"ts" toon:"ts"`
	// Line discriminator.
	Type  string      `json:"type" toon:"type"`
	Usage ExportUsage `json:"usage" toon:"usage"`
}

// ExportUsage is the token-usage sub-object on an ExportLine (llm_call events).
type ExportUsage struct {
	// Tokens written to the prompt cache.
	CacheCreation int64 `json:"cache_creation" toon:"cache_creation"`
	// Tokens served from the prompt cache.
	CacheRead int64 `json:"cache_read" toon:"cache_read"`
	// Prompt (input) tokens for the call.
	InputTokens int64 `json:"input_tokens" toon:"input_tokens"`
	// Generated (output) tokens for the call.
	OutputTokens int64 `json:"output_tokens" toon:"output_tokens"`
}
