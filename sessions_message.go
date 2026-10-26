package flashduty

import (
	"context"
	"fmt"
	"io"
)

// The session message, create, attachment-upload and subscribe endpoints are
// hand-written: they are not part of the vendored OpenAPI, and subscribe is a
// long-lived SSE stream the typed JSON path cannot model.

// SessionCreateRequest creates an empty session. It is needed before the first
// message when attachments must be uploaded first, because an attachment is
// stored under an existing session.
type SessionCreateRequest struct {
	// AppName is the agent app, e.g. "ai-sre". Required.
	AppName string `json:"app_name"`
	// EntryKind is how the session was initiated: web, im, api or automation.
	EntryKind string `json:"entry_kind,omitempty"`
	// TeamID binds the session to a team (0 = account scope). Immutable.
	TeamID int64 `json:"team_id,omitempty"`
}

// SessionCreateResponse is the created session.
type SessionCreateResponse struct {
	SessionID string `json:"session_id"`
	AppName   string `json:"app_name"`
	EntryKind string `json:"entry_kind"`
}

// WriteCreate creates a session.
//
// API: POST /safari/session/create.
func (s *SessionsService) WriteCreate(ctx context.Context, req *SessionCreateRequest) (*SessionCreateResponse, *Response, error) {
	out := new(SessionCreateResponse)
	resp, err := s.client.do(ctx, "/safari/session/create", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// SessionMessageRequest is one user turn sent to a session.
type SessionMessageRequest struct {
	// SessionID continues an existing session. Omit it to start a new one, in
	// which case AppName and EntryKind are required.
	SessionID string `json:"session_id,omitempty"`
	// AppName is the agent app ("ai-sre"); required when SessionID is empty.
	AppName string `json:"app_name,omitempty"`
	// EntryKind is how the run was initiated (web, im, api); required when
	// SessionID is empty.
	EntryKind string `json:"entry_kind,omitempty"`
	// Query is the message text (max 32000 characters).
	Query string `json:"query,omitempty"`
	// Parts carries additional message parts, such as uploaded files.
	Parts []SessionMessagePart `json:"parts,omitempty"`
	// Surface tags where this turn originated, e.g. "cli".
	Surface string `json:"surface,omitempty"`
	// DeliveryMode negotiates how replies are delivered. "reply_only" makes the
	// agent answer through its reply tool only; check the ack to see whether
	// the server accepted it.
	DeliveryMode string `json:"delivery_mode,omitempty"`
	// TeamID binds a new session to a team (0 = account scope).
	TeamID int64 `json:"team_id,omitempty"`
	// EnvironmentKind selects the execution environment: "cloud" or "byoc".
	EnvironmentKind string `json:"environment_kind,omitempty"`
	// EnvironmentID selects a specific environment of that kind.
	EnvironmentID string `json:"environment_id,omitempty"`
}

// SessionMessagePart is one part of a message: text or a stored file.
type SessionMessagePart struct {
	// Type is "text" or "file".
	Type string `json:"type"`
	// Text is set when Type is "text".
	Text string `json:"text,omitempty"`
	// File is set when Type is "file".
	File *SessionFilePart `json:"file,omitempty"`
}

// SessionFilePart references a file already stored by UploadAttachment.
type SessionFilePart struct {
	Mime      string `json:"mime"`
	Filename  string `json:"filename,omitempty"`
	ObjectKey string `json:"object_key"`
	Size      int64  `json:"size,omitempty"`
}

// SessionMessageAck is the acknowledgement of an accepted message. The turn
// runs asynchronously; follow it with Subscribe.
type SessionMessageAck struct {
	SessionID        string `json:"session_id"`
	InvocationID     string `json:"invocation_id"`
	RootInvocationID string `json:"root_invocation_id"`
	// DeliveryMode is the mode the server actually accepted. It is empty when
	// the requested mode was not granted.
	DeliveryMode string `json:"delivery_mode,omitempty"`
}

// WriteMessage sends a user turn and returns once the server has durably
// queued it (HTTP 202). Replies are not in the ack; read them from Subscribe.
//
// API: POST /safari/session/message.
func (s *SessionsService) WriteMessage(ctx context.Context, req *SessionMessageRequest) (*SessionMessageAck, *Response, error) {
	if req == nil {
		return nil, nil, fmt.Errorf("flashduty: SessionMessageRequest is required")
	}
	out := new(SessionMessageAck)
	resp, err := s.client.do(ctx, "/safari/session/message", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// SessionAttachmentUploadRequest carries one file to store under a session.
type SessionAttachmentUploadRequest struct {
	// SessionID is the existing session the file belongs to. Required.
	SessionID string
	// File is the file content. Required. Max 50 MB.
	File io.Reader
	// Filename is the multipart file name. Required.
	Filename string
}

// SessionAttachment describes a stored attachment.
type SessionAttachment struct {
	ObjectKey    string `json:"object_key"`
	Mime         string `json:"mime"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
	ExpiresIn    int    `json:"expires_in"`
	AttachmentID string `json:"attachment_id"`
}

// FilePart converts the stored attachment into a message part for
// SessionMessageRequest.Parts.
func (a *SessionAttachment) FilePart() SessionMessagePart {
	return SessionMessagePart{
		Type: "file",
		File: &SessionFilePart{Mime: a.Mime, Filename: a.Filename, ObjectKey: a.ObjectKey, Size: a.Size},
	}
}

// UploadAttachment stores a file under a session as a multipart upload.
//
// API: POST /safari/attachment/upload.
func (s *SessionsService) UploadAttachment(ctx context.Context, req *SessionAttachmentUploadRequest) (*SessionAttachment, *Response, error) {
	if req == nil || req.File == nil || req.SessionID == "" || req.Filename == "" {
		return nil, nil, fmt.Errorf("flashduty: SessionAttachmentUploadRequest requires SessionID, Filename and a non-nil File")
	}
	out := new(SessionAttachment)
	fields := map[string]string{"session_id": req.SessionID}
	resp, err := s.client.uploadFile(ctx, "/safari/attachment/upload", nil, fields, req.Filename, req.File, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}
