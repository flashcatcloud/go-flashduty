package flashduty

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAutomationTriggerWriteFireUsesBearerTokenAndDecodesAccepted(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/safari/automation/triggers/trig_abc/fire" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("app_key"); got != "" {
			t.Errorf("app_key must not be sent for trigger fire, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok_secret" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"text":"deployment finished"`) ||
			!strings.Contains(string(body), `"dedup_key":"deploy-1"`) {
			t.Errorf("body = %s", body)
		}

		w.Header().Set("Flashcat-Request-Id", "RIDF")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"request_id":"RIDF","data":{"run_id":"taskrun_1","rule_id":"auto_1","trigger_kind":"http_post","status":"running"}}`)
	})

	out, resp, err := c.Automations.TriggerWriteFire(context.Background(), "trig_abc", "tok_secret", &AutomationFireAPITriggerRequest{
		Text:     "deployment finished",
		DedupKey: "deploy-1",
	})
	if err != nil {
		t.Fatalf("TriggerWriteFire error: %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusAccepted || resp.RequestID != "RIDF" {
		t.Fatalf("response meta = %+v", resp)
	}
	if out == nil || out.RunID != "taskrun_1" || out.RuleID != "auto_1" || out.TriggerKind != "http_post" || out.Status != "running" {
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

func TestAutomationRuleUpdateRequestCanSendFalseValues(t *testing.T) {
	payload, err := json.Marshal(&AutomationRuleUpdateRequest{
		RuleID:                     "auto_1",
		Enabled:                    Bool(false),
		ScheduleTriggerEnabled:     Bool(false),
		HTTPPostTriggerEnabled:     Bool(false),
		RotateHTTPPostTriggerToken: false,
		TeamID:                     Int64(0),
		EnvironmentID:              String(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(payload)
	for _, want := range []string{
		`"rule_id":"auto_1"`,
		`"enabled":false`,
		`"schedule_trigger_enabled":false`,
		`"http_post_trigger_enabled":false`,
		`"team_id":0`,
		`"environment_id":""`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("payload %s missing %s", got, want)
		}
	}
	if strings.Contains(got, "rotate_http_post_trigger_token") {
		t.Fatalf("zero rotate flag should stay omitted: %s", got)
	}
}
