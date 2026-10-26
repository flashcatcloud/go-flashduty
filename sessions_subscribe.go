package flashduty

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
)

// SessionSubscribeRequest opens a session event stream. The stream first
// replays persisted events strictly newer than the (LastEventAt, LastEventID)
// cursor, then tails live frames.
type SessionSubscribeRequest struct {
	SessionID string `json:"session_id"`
	// LastEventAt is the resume cursor: a Unix-millisecond timestamp. 0 replays
	// from the start of the session.
	LastEventAt int64 `json:"last_event_at,omitempty"`
	// LastEventID breaks ties between events sharing LastEventAt.
	LastEventID string `json:"last_event_id,omitempty"`
}

// SessionEventFrame is one frame of a session event stream. Persisted events
// carry EventID and CreatedAt, with Type set to the event author ("user",
// "assistant", ...); ephemeral frames (turn_complete, assistant_token, ...)
// carry no EventID and Type is the frame type. Payload is the frame-specific
// JSON.
type SessionEventFrame struct {
	EventID          string          `json:"event_id,omitempty"`
	CreatedAt        int64           `json:"created_at"`
	Type             string          `json:"type"`
	AgentPath        string          `json:"agent_path,omitempty"`
	Payload          json.RawMessage `json:"payload"`
	InvocationID     string          `json:"invocation_id,omitempty"`
	RootInvocationID string          `json:"root_invocation_id,omitempty"`
	DeliveryMode     string          `json:"delivery_mode,omitempty"`
	IsError          bool            `json:"is_error,omitempty"`
	ErrorCode        string          `json:"error_code,omitempty"`
}

// Subscribe opens POST /safari/session/subscribe and returns the live SSE
// response body. The caller owns it and MUST Close it; read frames with
// NewSessionFrameReader. The stream is unbounded in time (the server closes it
// after an idle period), so it is served by a client with no overall timeout:
// bound its lifetime with ctx.
//
// On any non-2xx status the body is the usual JSON error envelope: Subscribe
// reads and closes it and returns a typed error with a nil ReadCloser.
//
// API: POST /safari/session/subscribe.
func (s *SessionsService) Subscribe(ctx context.Context, req *SessionSubscribeRequest) (io.ReadCloser, *Response, error) {
	// The shared client's Timeout also bounds reading the body, which would cut
	// a long-lived stream. Copy it without the overall timeout; the transport
	// and redirect policy are shared.
	streamClient := *s.client.client
	streamClient.Timeout = 0
	return s.client.openStream(ctx, &streamClient, "/safari/session/subscribe", "text/event-stream", req)
}

// SessionStreamError is a server-reported error carried in-band on the stream
// as an "event: error" record.
type SessionStreamError struct {
	Message string
}

func (e *SessionStreamError) Error() string {
	return "flashduty: session stream error: " + e.Message
}

// SessionFrameReader decodes the SSE stream returned by Subscribe into frames.
type SessionFrameReader struct {
	sc *bufio.Scanner
}

// NewSessionFrameReader wraps a Subscribe stream. Heartbeat comments and
// undecodable records are skipped.
//
//	rc, _, err := client.Sessions.Subscribe(ctx, req)
//	if err != nil { return err }
//	defer rc.Close()
//	fr := flashduty.NewSessionFrameReader(rc)
//	for {
//	    frame, err := fr.Next()
//	    if err == io.EOF { break } // server finished the stream
//	    if err != nil { return err }
//	    // ... handle frame ...
//	}
func NewSessionFrameReader(r io.Reader) *SessionFrameReader {
	sc := bufio.NewScanner(r)
	// One persisted frame can carry a large tool output.
	sc.Buffer(make([]byte, 0, 64*1024), maxResponseBodySize)
	return &SessionFrameReader{sc: sc}
}

// Next returns the next frame. It returns io.EOF when the server ends the
// stream (the "[DONE]" sentinel or a clean close), a *SessionStreamError for an
// in-band error record, and the transport error (including context
// cancellation) otherwise.
func (r *SessionFrameReader) Next() (*SessionEventFrame, error) {
	eventName := ""
	for r.sc.Scan() {
		line := strings.TrimRight(r.sc.Text(), "\r")
		switch {
		case line == "":
			eventName = ""
		case strings.HasPrefix(line, ":"):
			// heartbeat comment
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")
			if eventName == "error" {
				msg := data
				var s string
				if json.Unmarshal([]byte(data), &s) == nil {
					msg = s
				}
				return nil, &SessionStreamError{Message: msg}
			}
			if data == "[DONE]" {
				return nil, io.EOF
			}
			var frame SessionEventFrame
			if err := json.Unmarshal([]byte(data), &frame); err != nil || frame.Type == "" {
				continue
			}
			return &frame, nil
		}
	}
	if err := r.sc.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}
