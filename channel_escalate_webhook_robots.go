package flashduty

import "context"

// ChannelsChannelEscalateWebhookRobotListRequest filters the webhook robots
// referenced by channel escalation rules.
type ChannelsChannelEscalateWebhookRobotListRequest struct {
	Query string `json:"query,omitempty" toon:"query,omitempty"`
	Type  string `json:"type,omitempty" toon:"type,omitempty"`
}

// ChannelsChannelEscalateWebhookRobotListResponse is the robot list result.
type ChannelsChannelEscalateWebhookRobotListResponse struct {
	List []ChannelsChannelEscalateWebhookRobotListResponseListItem `json:"list" toon:"list"`
}

// ChannelsChannelEscalateWebhookRobotListResponseListItem is one configured
// escalation webhook robot.
type ChannelsChannelEscalateWebhookRobotListResponseListItem struct {
	ReferencedBy []ChannelsChannelEscalateWebhookRobotListResponseListItemReferencedByItem `json:"referenced_by" toon:"referenced_by"`
	Settings     map[string]any                                                            `json:"settings" toon:"settings"`
	Type         string                                                                    `json:"type" toon:"type"`
}

// ChannelsChannelEscalateWebhookRobotListResponseListItemReferencedByItem identifies an escalation rule that uses a robot.
type ChannelsChannelEscalateWebhookRobotListResponseListItemReferencedByItem struct {
	ChannelID        int64  `json:"channel_id" toon:"channel_id"`
	ChannelName      string `json:"channel_name" toon:"channel_name"`
	EscalateRuleID   string `json:"escalate_rule_id" toon:"escalate_rule_id"`
	EscalateRuleName string `json:"escalate_rule_name" toon:"escalate_rule_name"`
}

// ChannelEscalateWebhookRobotList lists webhook robots configured in
// escalation rules.
//
// API: POST /channel/escalate/webhook/robot/list
func (s *ChannelsService) ChannelEscalateWebhookRobotList(ctx context.Context, req *ChannelsChannelEscalateWebhookRobotListRequest) (*ChannelsChannelEscalateWebhookRobotListResponse, *Response, error) {
	out := new(ChannelsChannelEscalateWebhookRobotListResponse)
	resp, err := s.client.do(ctx, "/channel/escalate/webhook/robot/list", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}
