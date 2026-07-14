package flashduty

import "context"

// AutomationRuleRunPreflight describes the checks performed before a manual
// Automation run starts.
type AutomationRuleRunPreflight struct {
	AppName  string   `json:"app_name" toon:"app_name"`
	Checks   []string `json:"checks" toon:"checks"`
	OK       bool     `json:"ok" toon:"ok"`
	OwnerID  int64    `json:"owner_id" toon:"owner_id"`
	Scope    string   `json:"scope" toon:"scope"`
	TeamID   int64    `json:"team_id" toon:"team_id"`
	Warnings []string `json:"warnings" toon:"warnings"`
}

// AutomationRuleRunView identifies the Automation run and its optional
// AI-SRE session.
type AutomationRuleRunView struct {
	RunID     string `json:"run_id" toon:"run_id"`
	SessionID string `json:"session_id" toon:"session_id"`
}

// AutomationRuleRunResponse is returned after starting an Automation rule.
type AutomationRuleRunResponse struct {
	Preflight   AutomationRuleRunPreflight `json:"preflight" toon:"preflight"`
	RuleID      string                     `json:"rule_id" toon:"rule_id"`
	Run         AutomationRuleRunView      `json:"run" toon:"run"`
	TriggerKind string                     `json:"trigger_kind" toon:"trigger_kind"`
}

// RuleWriteRun starts a manual run for an Automation rule.
//
// API: POST /safari/automation/rule/run
func (s *AutomationsService) RuleWriteRun(ctx context.Context, req *AutomationRuleIDRequest) (*AutomationRuleRunResponse, *Response, error) {
	out := new(AutomationRuleRunResponse)
	resp, err := s.client.do(ctx, "/safari/automation/rule/run", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}
