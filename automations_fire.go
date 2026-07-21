package flashduty

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AutomationFireAPITriggerRequest is the payload accepted by an Automation
// HTTP POST trigger.
type AutomationFireAPITriggerRequest struct {
	// Context text passed to this Automation run.
	Text string `json:"text,omitempty" toon:"text,omitempty"`
}

// AutomationFireAPITriggerResponse is the result returned by an Automation
// HTTP POST trigger.
type AutomationFireAPITriggerResponse struct {
	// Result type. The API-trigger success path returns routine_fire.
	Type string `json:"type" toon:"type"`
	// Started session ID returned by the API-trigger success path.
	SessionID string `json:"session_id" toon:"session_id"`
	// Console URL for the started session.
	SessionURL string `json:"session_url" toon:"session_url"`
}

// TriggerWriteFire triggers an Automation run through its HTTP POST trigger URL.
//
// This endpoint authenticates with the trigger's one-time bearer token rather
// than the account app_key used by the generated API methods.
//
// API: POST /safari/automation/triggers/{trigger_id}/fire (automation-trigger-write-fire).
func (s *AutomationsService) TriggerWriteFire(ctx context.Context, triggerID, token string, req *AutomationFireAPITriggerRequest) (*AutomationFireAPITriggerResponse, *Response, error) {
	triggerID = strings.TrimSpace(triggerID)
	if triggerID == "" {
		return nil, nil, fmt.Errorf("flashduty: automation trigger_id is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil, fmt.Errorf("flashduty: automation trigger token is required")
	}

	path := "/safari/automation/triggers/" + url.PathEscape(triggerID) + "/fire"
	out := new(AutomationFireAPITriggerResponse)
	resp, err := s.client.doMethodWithoutAppKey(ctx, http.MethodPost, path, req, out, func(httpReq *http.Request) {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	})
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}
