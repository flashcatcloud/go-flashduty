package flashduty

import (
	"context"
	"fmt"
)

// The IM self-chat list endpoints are hand-written: they use gateway Auth
// "service" (app_key-callable, excluded from the public OpenAPI spec) and let
// callers discover chat/channel IDs for an IM integration (e.g. when building
// escalation rules).

// DingtalkSelfChatListRequest lists DingTalk group chats the integration's bot
// has been installed into.
type DingtalkSelfChatListRequest struct {
	// DataSourceID is the DingTalk IM integration ID (integration_id / data_source_id). Required.
	DataSourceID int64 `json:"data_source_id" toon:"data_source_id"`
}

// DingtalkSelfChatItem is one DingTalk group chat returned by
// POST /datasource/dingtalk/self-chat/list.
type DingtalkSelfChatItem struct {
	Name   string `json:"name" toon:"name"`
	ChatID string `json:"chat_id" toon:"chat_id"`
	Origin string `json:"origin,omitempty" toon:"origin,omitempty"`
}

// DingtalkSelfChatListResponse is the response from
// POST /datasource/dingtalk/self-chat/list.
type DingtalkSelfChatListResponse struct {
	Items []DingtalkSelfChatItem `json:"items" toon:"items"`
}

// DingtalkSelfChatList lists DingTalk chats where the integration's bot is
// installed.
//
// API: POST /datasource/dingtalk/self-chat/list.
func (s *ImIntegrationsService) DingtalkSelfChatList(ctx context.Context, req *DingtalkSelfChatListRequest) (*DingtalkSelfChatListResponse, *Response, error) {
	if req == nil || req.DataSourceID <= 0 {
		return nil, nil, fmt.Errorf("flashduty: data_source_id must be positive")
	}
	out := new(DingtalkSelfChatListResponse)
	resp, err := s.client.do(ctx, "/datasource/dingtalk/self-chat/list", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// FeishuSelfChatListRequest lists Feishu (Lark) chats the integration's bot
// belongs to.
type FeishuSelfChatListRequest struct {
	// DataSourceID is the Feishu IM integration ID (integration_id / data_source_id). Required.
	DataSourceID int64 `json:"data_source_id" toon:"data_source_id"`
}

// FeishuSelfChatItem is one Feishu group chat returned by
// POST /datasource/feishu/self-chat/list.
type FeishuSelfChatItem struct {
	ChatID      string `json:"chat_id" toon:"chat_id"`
	Avatar      string `json:"avatar,omitempty" toon:"avatar,omitempty"`
	Name        string `json:"name" toon:"name"`
	Description string `json:"description,omitempty" toon:"description,omitempty"`
	OwnerID     string `json:"owner_id,omitempty" toon:"owner_id,omitempty"`
	OwnerIDType string `json:"owner_id_type,omitempty" toon:"owner_id_type,omitempty"`
	External    bool   `json:"external,omitempty" toon:"external,omitempty"`
	TenantKey   string `json:"tenant_key,omitempty" toon:"tenant_key,omitempty"`
	Origin      string `json:"origin,omitempty" toon:"origin,omitempty"`
}

// FeishuSelfChatListResponse is the response from
// POST /datasource/feishu/self-chat/list.
type FeishuSelfChatListResponse struct {
	Items []FeishuSelfChatItem `json:"items" toon:"items"`
}

// FeishuSelfChatList lists Feishu (Lark) chats the integration's bot belongs
// to.
//
// API: POST /datasource/feishu/self-chat/list.
func (s *ImIntegrationsService) FeishuSelfChatList(ctx context.Context, req *FeishuSelfChatListRequest) (*FeishuSelfChatListResponse, *Response, error) {
	if req == nil || req.DataSourceID <= 0 {
		return nil, nil, fmt.Errorf("flashduty: data_source_id must be positive")
	}
	out := new(FeishuSelfChatListResponse)
	resp, err := s.client.do(ctx, "/datasource/feishu/self-chat/list", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// SlackSelfChatListRequest lists Slack channels visible to the integration.
type SlackSelfChatListRequest struct {
	// DataSourceID is the Slack IM integration ID (integration_id / data_source_id). Required.
	DataSourceID int64 `json:"data_source_id" toon:"data_source_id"`
	// ForceRefresh bypasses the server-side Slack channel cache when true.
	ForceRefresh bool `json:"force_refresh,omitempty" toon:"force_refresh,omitempty"`
}

// SlackSelfChatItem is one Slack channel returned by
// POST /datasource/slack/self-chat/list.
type SlackSelfChatItem struct {
	ID         string `json:"id" toon:"id"`
	Name       string `json:"name" toon:"name"`
	IsChannel  bool   `json:"is_channel,omitempty" toon:"is_channel,omitempty"`
	IsGroup    bool   `json:"is_group,omitempty" toon:"is_group,omitempty"`
	IsPrivate  bool   `json:"is_private,omitempty" toon:"is_private,omitempty"`
	IsArchived bool   `json:"is_archived,omitempty" toon:"is_archived,omitempty"`
	IsGeneral  bool   `json:"is_general,omitempty" toon:"is_general,omitempty"`
	IsMember   bool   `json:"is_member,omitempty" toon:"is_member,omitempty"`
	NumMembers int    `json:"num_members,omitempty" toon:"num_members,omitempty"`
	Origin     string `json:"origin,omitempty" toon:"origin,omitempty"`
}

// SlackSelfChatListResponse is the response from
// POST /datasource/slack/self-chat/list.
type SlackSelfChatListResponse struct {
	Items []SlackSelfChatItem `json:"items" toon:"items"`
}

// SlackSelfChatList lists Slack channels visible to the integration.
//
// API: POST /datasource/slack/self-chat/list.
func (s *ImIntegrationsService) SlackSelfChatList(ctx context.Context, req *SlackSelfChatListRequest) (*SlackSelfChatListResponse, *Response, error) {
	if req == nil || req.DataSourceID <= 0 {
		return nil, nil, fmt.Errorf("flashduty: data_source_id must be positive")
	}
	out := new(SlackSelfChatListResponse)
	resp, err := s.client.do(ctx, "/datasource/slack/self-chat/list", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}

// TeamsSelfChatListRequest lists Microsoft Teams channels linked to the
// integration.
type TeamsSelfChatListRequest struct {
	// DataSourceID is the Teams IM integration ID (integration_id / data_source_id). Required.
	DataSourceID int64 `json:"data_source_id" toon:"data_source_id"`
	// TeamID optionally restricts the result to one linked Microsoft Teams team.
	TeamID string `json:"team_id,omitempty" toon:"team_id,omitempty"`
	// ForceRefresh bypasses the server-side channel cache when true.
	ForceRefresh bool `json:"force_refresh,omitempty" toon:"force_refresh,omitempty"`
}

// TeamsChannelItem is one channel inside a linked Microsoft Teams team.
type TeamsChannelItem struct {
	ChannelID   string `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	ChannelName string `json:"channel_name,omitempty" toon:"channel_name,omitempty"`
	Origin      string `json:"origin,omitempty" toon:"origin,omitempty"`
}

// TeamsTeamChannelsItem groups channels under one linked Microsoft Teams team.
type TeamsTeamChannelsItem struct {
	TeamID   string             `json:"team_id,omitempty" toon:"team_id,omitempty"`
	TeamName string             `json:"team_name,omitempty" toon:"team_name,omitempty"`
	Channels []TeamsChannelItem `json:"channels,omitempty" toon:"channels,omitempty"`
}

// TeamsSelfChatListResponse is the response from
// POST /datasource/teams/self-chat/list.
type TeamsSelfChatListResponse struct {
	Items []TeamsTeamChannelsItem `json:"items" toon:"items"`
}

// TeamsSelfChatList lists Microsoft Teams channels linked to the integration.
//
// API: POST /datasource/teams/self-chat/list.
func (s *ImIntegrationsService) TeamsSelfChatList(ctx context.Context, req *TeamsSelfChatListRequest) (*TeamsSelfChatListResponse, *Response, error) {
	if req == nil || req.DataSourceID <= 0 {
		return nil, nil, fmt.Errorf("flashduty: data_source_id must be positive")
	}
	out := new(TeamsSelfChatListResponse)
	resp, err := s.client.do(ctx, "/datasource/teams/self-chat/list", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}
