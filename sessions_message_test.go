package flashduty

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSessionsWriteMessageSendsRequestAndDecodesAck(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/safari/session/message" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("app_key"); got != "KEY" {
			t.Errorf("app_key = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["surface"] != "cli" || body["delivery_mode"] != "reply_only" || body["query"] != "hi" {
			t.Errorf("body = %v", body)
		}
		parts, _ := body["parts"].([]any)
		if len(parts) != 1 {
			t.Fatalf("parts = %v", body["parts"])
		}
		file := parts[0].(map[string]any)["file"].(map[string]any)
		if file["object_key"] != "attachments/a/b.txt" || file["mime"] != "text/plain" {
			t.Errorf("file part = %v", file)
		}
		if _, ok := body["team_id"]; ok {
			t.Errorf("zero team_id must be omitted: %v", body)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"request_id":"r1","data":{"session_id":"s1","invocation_id":"i1","root_invocation_id":"i1","delivery_mode":"reply_only"}}`)
	})

	ack, _, err := c.Sessions.WriteMessage(context.Background(), &SessionMessageRequest{
		AppName:      "ai-sre",
		EntryKind:    "api",
		Query:        "hi",
		Surface:      "cli",
		DeliveryMode: "reply_only",
		Parts: []SessionMessagePart{{
			Type: "file",
			File: &SessionFilePart{Mime: "text/plain", Filename: "b.txt", ObjectKey: "attachments/a/b.txt", Size: 3},
		}},
	})
	if err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	if ack.SessionID != "s1" || ack.InvocationID != "i1" || ack.RootInvocationID != "i1" || ack.DeliveryMode != "reply_only" {
		t.Errorf("ack = %+v", ack)
	}
}

func TestSessionsWriteMessageSurfacesAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"request_id":"r1","error":{"code":"InvalidParameter","message":"bad team"}}`)
	})
	_, _, err := c.Sessions.WriteMessage(context.Background(), &SessionMessageRequest{AppName: "ai-sre"})
	if err == nil || !strings.Contains(err.Error(), "bad team") {
		t.Fatalf("err = %v", err)
	}
}

func TestSessionsWriteCreate(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/safari/session/create" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body SessionCreateRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.AppName != "ai-sre" || body.EntryKind != "api" || body.TeamID != 7 {
			t.Errorf("body = %+v", body)
		}
		_, _ = io.WriteString(w, `{"request_id":"r1","data":{"session_id":"s9","app_name":"ai-sre","entry_kind":"api"}}`)
	})
	out, _, err := c.Sessions.WriteCreate(context.Background(), &SessionCreateRequest{AppName: "ai-sre", EntryKind: "api", TeamID: 7})
	if err != nil || out.SessionID != "s9" {
		t.Fatalf("out = %+v, err = %v", out, err)
	}
}

func TestSessionsUploadAttachmentMultipart(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/safari/attachment/upload" {
			t.Errorf("path = %s", r.URL.Path)
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseMultipartForm(1 << 20); err != nil { //nolint:gosec // test server; body capped by MaxBytesReader above
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("session_id"); got != "s1" {
			t.Errorf("session_id = %q", got)
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = f.Close() }()
		data, _ := io.ReadAll(f)
		if hdr.Filename != "notes.txt" || string(data) != "hello" {
			t.Errorf("file = %q %q", hdr.Filename, data)
		}
		_, _ = io.WriteString(w, `{"request_id":"r1","data":{"object_key":"attachments/att_1/notes.txt","mime":"text/plain","filename":"notes.txt","size":5,"url":"https://x/y","expires_in":3600,"attachment_id":"att_1"}}`)
	})

	att, _, err := c.Sessions.UploadAttachment(context.Background(), &SessionAttachmentUploadRequest{
		SessionID: "s1", Filename: "notes.txt", File: strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}
	part := att.FilePart()
	if part.Type != "file" || part.File.ObjectKey != "attachments/att_1/notes.txt" || part.File.Mime != "text/plain" || part.File.Size != 5 || part.File.Filename != "notes.txt" {
		t.Errorf("part = %+v", part.File)
	}
}

func TestSessionsUploadAttachmentValidatesInput(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	for _, req := range []*SessionAttachmentUploadRequest{
		nil,
		{Filename: "a", File: strings.NewReader("x")},
		{SessionID: "s", File: strings.NewReader("x")},
		{SessionID: "s", Filename: "a"},
	} {
		if _, _, err := c.Sessions.UploadAttachment(context.Background(), req); err == nil {
			t.Errorf("expected error for %+v", req)
		}
	}
}
