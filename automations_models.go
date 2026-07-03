package flashduty

// AutomationRuleUpdateRequest updates mutable fields on an Automation rule.
// Pointer fields preserve partial-update semantics: nil means leave unchanged,
// while a non-nil zero value is sent to the API.
type AutomationRuleUpdateRequest struct {
	// Target rule ID.
	RuleID string `json:"rule_id,omitempty" toon:"rule_id,omitempty"`
	// New rule name.
	Name *string `json:"name,omitempty" toon:"name,omitempty"`
	// Only the current value is accepted; personal/team scope is immutable after creation.
	TeamID *int64 `json:"team_id,omitempty" toon:"team_id,omitempty"`
	// Whether the rule is enabled.
	Enabled *bool `json:"enabled,omitempty" toon:"enabled,omitempty"`
	// Run cadence. Supports 4 fields (`hour day month weekday`, minute defaults to 0) and 5 fields (`minute hour day month weekday`).
	CronExpr *string `json:"cron_expr,omitempty" toon:"cron_expr,omitempty"`
	// Whether the schedule trigger is enabled.
	ScheduleTriggerEnabled *bool `json:"schedule_trigger_enabled,omitempty" toon:"schedule_trigger_enabled,omitempty"`
	// New task prompt.
	Prompt *string `json:"prompt,omitempty" toon:"prompt,omitempty"`
	// Runtime environment kind. Omit or send an empty value for automatic selection.
	EnvironmentKind *string `json:"environment_kind,omitempty" toon:"environment_kind,omitempty"`
	// BYOC Runner ID.
	EnvironmentID *string `json:"environment_id,omitempty" toon:"environment_id,omitempty"`
	// Whether the HTTP POST trigger is enabled. Sending true creates one when missing.
	HTTPPostTriggerEnabled *bool `json:"http_post_trigger_enabled,omitempty" toon:"http_post_trigger_enabled,omitempty"`
	// Whether to rotate the HTTP POST trigger token. The new token is returned only in this response.
	RotateHTTPPostTriggerToken bool `json:"rotate_http_post_trigger_token,omitempty" toon:"rotate_http_post_trigger_token,omitempty"`
	// Whether the on-call incident trigger is enabled. Sending true creates it when missing and channel/severity filters are provided.
	OncallIncidentTriggerEnabled *bool `json:"oncall_incident_trigger_enabled,omitempty" toon:"oncall_incident_trigger_enabled,omitempty"`
	// On-call channel IDs whose new incidents can trigger this rule.
	OncallIncidentChannelIDs *[]int64 `json:"oncall_incident_channel_ids,omitempty" toon:"oncall_incident_channel_ids,omitempty"`
	// Incident severities that can trigger this rule.
	OncallIncidentSeverities *[]string `json:"oncall_incident_severities,omitempty" toon:"oncall_incident_severities,omitempty"`
}
