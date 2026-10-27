package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestDedicatedIntegrationRequests(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		call func(*Client) error
	}{
		{"list", `{"channel_id":20001,"p":2,"limit":10,"name":"Primary","plugin_type":"standard.alert","status":"enabled"}`, func(c *Client) error {
			_, _, err := c.DedicatedIntegrations.ReadList(context.Background(), &ChannelIntegrationListRequest{ListOptions: ListOptions{Page: 2, Limit: 10}, ChannelID: 20001, Name: "Primary", PluginType: "standard.alert", Status: "enabled"})
			return err
		}},
		{"info", `{"integration_id":10001}`, func(c *Client) error {
			_, _, err := c.DedicatedIntegrations.ReadInfo(context.Background(), &ChannelIntegrationIDRequest{IntegrationID: 10001})
			return err
		}},
		{"create", `{"channel_id":20001,"plugin_type":"standard.alert","name":"Primary","integration_key":"0123456789abcdef0123456789abcdef100","settings":{"headers":{}}}`, func(c *Client) error {
			_, _, err := c.DedicatedIntegrations.WriteCreate(context.Background(), &ChannelIntegrationCreateRequest{ChannelID: 20001, PluginType: "standard.alert", Name: "Primary", IntegrationKey: "0123456789abcdef0123456789abcdef100", Settings: map[string]any{"headers": map[string]any{}}})
			return err
		}},
		{"update", `{"integration_id":10001,"description":""}`, func(c *Client) error {
			empty := ""
			_, _, err := c.DedicatedIntegrations.WriteUpdate(context.Background(), &ChannelIntegrationUpdateRequest{IntegrationID: 10001, Description: &empty})
			return err
		}},
		{"delete", `{"integration_id":10001}`, func(c *Client) error {
			_, err := c.DedicatedIntegrations.WriteDelete(context.Background(), &ChannelIntegrationIDRequest{IntegrationID: 10001})
			return err
		}},
		{"enable", `{"integration_id":10001}`, func(c *Client) error {
			_, err := c.DedicatedIntegrations.WriteEnable(context.Background(), &ChannelIntegrationIDRequest{IntegrationID: 10001})
			return err
		}},
		{"disable", `{"integration_id":10001}`, func(c *Client) error {
			_, err := c.DedicatedIntegrations.WriteDisable(context.Background(), &ChannelIntegrationIDRequest{IntegrationID: 10001})
			return err
		}},
		{"key/rotate", `{"integration_id":10001}`, func(c *Client) error {
			_, _, err := c.DedicatedIntegrations.WriteRotateKey(context.Background(), &ChannelIntegrationIDRequest{IntegrationID: 10001})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/channel/integration/"+tc.name || r.URL.Query().Get("app_key") != "KEY" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				var want, got map[string]any
				err := json.Unmarshal([]byte(tc.body), &want)
				if err != nil {
					t.Error(err)
				}
				err = json.NewDecoder(r.Body).Decode(&got)
				if err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("body = %#v, want %#v", got, want)
				}
				_, _ = w.Write([]byte(`{"data":{}}`))
			})
			err := tc.call(c)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDedicatedIntegrationUpdatePreservesEmptySettings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings map[string]any
		present  bool
	}{
		{"omitted", nil, false},
		{"empty replacement", map[string]any{}, true},
		{"masked secret", map[string]any{"password": "******"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(&ChannelIntegrationUpdateRequest{IntegrationID: 10001, Settings: tc.settings})
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			_ = json.Unmarshal(body, &got)
			_, present := got["settings"]
			if present != tc.present {
				t.Fatalf("settings presence = %v, want %v: %s", present, tc.present, body)
			}
			if present && !reflect.DeepEqual(got["settings"], tc.settings) {
				t.Fatalf("settings changed: %s", body)
			}
		})
	}
}

func TestDedicatedIntegrationResponsePreservesOwnershipAndConfiguration(t *testing.T) {
	const item = `{"integration_id":10001,"channel_id":20001,"integration_key":"fixture-key","exclusive_data_source_id":0,"plugin_id":9,"plugin_type":"standard.alert","plugin_type_name":"Standard","category":"event.alert","name":"Primary","description":"","status":"enabled","created_at":1700000000,"updated_at":1700000001,"last_time":0,"last_source_time":0,"settings":{"password":"******"}}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"items":[` + item + `],"total":1,"p":1,"limit":100}}`))
	})
	out, _, err := c.DedicatedIntegrations.ReadList(context.Background(), &ChannelIntegrationListRequest{ChannelID: 20001})
	if err != nil {
		t.Fatal(err)
	}
	if out.Total != 1 || len(out.Items) != 1 {
		t.Fatalf("unexpected page: %#v", out)
	}
	encoded, err := json.Marshal(out.Items[0])
	if err != nil {
		t.Fatal(err)
	}
	var want, got map[string]any
	_ = json.Unmarshal([]byte(item), &want)
	_ = json.Unmarshal(encoded, &got)
	if out.Items[0].CreatedAt.Unix() != 1700000000 || out.Items[0].UpdatedAt.Unix() != 1700000001 {
		t.Fatalf("timestamps changed: %#v", out.Items[0])
	}
	for _, field := range []string{"created_at", "updated_at"} {
		delete(want, field)
		delete(got, field)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response fields changed: got %s, want %s", encoded, item)
	}
}

func TestDedicatedIntegrationForbiddenDoesNotCallPublicAPI(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/channel/integration/info" {
			t.Errorf("unexpected fallback: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"AccessDenied","message":"Channel access denied"}}`))
	})
	_, _, err := c.DedicatedIntegrations.ReadInfo(context.Background(), &ChannelIntegrationIDRequest{IntegrationID: 10001})
	if ErrorCodeOf(err) != ErrorCodeAccessDenied || calls != 1 {
		t.Fatalf("error = %v, calls = %d", err, calls)
	}
}
