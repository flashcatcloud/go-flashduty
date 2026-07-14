package flashduty

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAutomationTriggerWriteFireUsesBearerToken(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/safari/automation/triggers/trig_abc/fire" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("app_key"); got != "" {
			t.Errorf("app_key must not be sent, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok_secret" {
			t.Errorf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"text":"deployment finished"`) {
			t.Errorf("body = %s", body)
		}
		w.Header().Set("Flashcat-Request-Id", "RIDF")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"request_id":"RIDF","data":{"type":"routine_fire","session_id":"sess_1","session_url":"/safari/session/sess_1"}}`)
	})

	out, resp, err := c.Automations.TriggerWriteFire(context.Background(), "trig_abc", "tok_secret", &AutomationFireAPITriggerRequest{Text: "deployment finished"})
	if err != nil {
		t.Fatalf("TriggerWriteFire error: %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK || resp.RequestID != "RIDF" {
		t.Fatalf("response meta = %+v", resp)
	}
	if out == nil || out.Type != "routine_fire" || out.SessionID != "sess_1" || out.SessionURL != "/safari/session/sess_1" {
		t.Fatalf("decoded response = %+v", out)
	}
}

func TestAutomationTriggerWriteFireValidatesInputs(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request should not be sent")
	})

	if _, _, err := c.Automations.TriggerWriteFire(context.Background(), "", "tok", nil); err == nil {
		t.Fatal("expected empty trigger_id error")
	}
	if _, _, err := c.Automations.TriggerWriteFire(context.Background(), "trig_abc", "", nil); err == nil {
		t.Fatal("expected empty token error")
	}
}
