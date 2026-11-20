package flashduty

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestImIntegrationsDingtalkSelfChatList(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/datasource/dingtalk/self-chat/list" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("app_key"); got != "KEY" {
			t.Errorf("app_key = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["data_source_id"] != float64(6100259006783) {
			t.Errorf("data_source_id = %v", body["data_source_id"])
		}
		_, _ = io.WriteString(w, `{"request_id":"r1","data":{"items":[{"name":"国际公司告警群","chat_id":"cid_abc123","origin":"user"}]}}`)
	})

	out, _, err := c.ImIntegrations.DingtalkSelfChatList(context.Background(), &DingtalkSelfChatListRequest{
		DataSourceID: 6100259006783,
	})
	if err != nil {
		t.Fatalf("DingtalkSelfChatList: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].ChatID != "cid_abc123" || out.Items[0].Name != "国际公司告警群" || out.Items[0].Origin != "user" {
		t.Errorf("items = %+v", out.Items)
	}
}

func TestImIntegrationsFeishuSelfChatList(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/datasource/feishu/self-chat/list" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["data_source_id"] != float64(42) {
			t.Errorf("data_source_id = %v", body["data_source_id"])
		}
		_, _ = io.WriteString(w, `{"request_id":"r2","data":{"items":[{"chat_id":"oc_123","name":"SRE Ops","external":false,"origin":"user"}]}}`)
	})

	out, _, err := c.ImIntegrations.FeishuSelfChatList(context.Background(), &FeishuSelfChatListRequest{
		DataSourceID: 42,
	})
	if err != nil {
		t.Fatalf("FeishuSelfChatList: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].ChatID != "oc_123" || out.Items[0].Name != "SRE Ops" || out.Items[0].Origin != "user" {
		t.Errorf("items = %+v", out.Items)
	}
}

func TestImIntegrationsSlackSelfChatList(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/datasource/slack/self-chat/list" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["data_source_id"] != float64(77) || body["force_refresh"] != true {
			t.Errorf("body = %v", body)
		}
		_, _ = io.WriteString(w, `{"request_id":"r3","data":{"items":[{"id":"C0123","name":"alerts","is_private":true,"origin":"user"}]}}`)
	})

	out, _, err := c.ImIntegrations.SlackSelfChatList(context.Background(), &SlackSelfChatListRequest{
		DataSourceID: 77,
		ForceRefresh: true,
	})
	if err != nil {
		t.Fatalf("SlackSelfChatList: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].ID != "C0123" || out.Items[0].Name != "alerts" || !out.Items[0].IsPrivate {
		t.Errorf("items = %+v", out.Items)
	}
}

func TestImIntegrationsTeamsSelfChatList(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/datasource/teams/self-chat/list" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["data_source_id"] != float64(88) || body["team_id"] != "t-1" || body["force_refresh"] != true {
			t.Errorf("body = %v", body)
		}
		_, _ = io.WriteString(w, `{"request_id":"r4","data":{"items":[{"team_id":"t-1","team_name":"Ops","channels":[{"channel_id":"19:ch1","channel_name":"General","origin":"user"}]}]}}`)
	})

	out, _, err := c.ImIntegrations.TeamsSelfChatList(context.Background(), &TeamsSelfChatListRequest{
		DataSourceID: 88,
		TeamID:       "t-1",
		ForceRefresh: true,
	})
	if err != nil {
		t.Fatalf("TeamsSelfChatList: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TeamID != "t-1" || len(out.Items[0].Channels) != 1 || out.Items[0].Channels[0].ChannelID != "19:ch1" {
		t.Errorf("items = %+v", out.Items)
	}
}

func TestImIntegrationsSelfChatListRequiresPositiveDataSourceID(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request to %s", r.URL.Path)
	})

	if _, _, err := c.ImIntegrations.DingtalkSelfChatList(context.Background(), nil); err == nil {
		t.Error("DingtalkSelfChatList(nil) expected error")
	}
	if _, _, err := c.ImIntegrations.DingtalkSelfChatList(context.Background(), &DingtalkSelfChatListRequest{}); err == nil {
		t.Error("DingtalkSelfChatList(0) expected error")
	}
	if _, _, err := c.ImIntegrations.FeishuSelfChatList(context.Background(), &FeishuSelfChatListRequest{}); err == nil {
		t.Error("FeishuSelfChatList(0) expected error")
	}
	if _, _, err := c.ImIntegrations.SlackSelfChatList(context.Background(), &SlackSelfChatListRequest{}); err == nil {
		t.Error("SlackSelfChatList(0) expected error")
	}
	if _, _, err := c.ImIntegrations.TeamsSelfChatList(context.Background(), &TeamsSelfChatListRequest{}); err == nil {
		t.Error("TeamsSelfChatList(0) expected error")
	}
}
