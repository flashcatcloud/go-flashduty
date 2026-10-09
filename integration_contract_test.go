package flashduty

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestIntegrationResponseContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target any
		body   string
	}{
		{"list item", new(IntegrationItem), `{"integration_id":42,"plugin_id":9,"account_id":100,"data_source_id":42,"exclusive_data_source_id":0,"integration_key":"test-key","settings":{"password":"test-password"},"editable":true,"creator_id":11,"updated_by":12,"no_editable":false}`},
		{"detail", new(IntegrationDetail), `{"integration_id":42,"plugin_id":9,"integration_key":"test-key","settings":{"headers":{"Authorization":"test-token"}},"editable":true,"creator_id":11,"updated_by":12,"no_editable":false,"test":{"success":false,"message":"test connection failed"}}`},
		{"create", new(CreateIntegrationResponse), `{"integration_id":42,"integration_key":"test-key","test":{"success":false,"message":"test connection failed"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(tc.body), tc.target); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(tc.target)
			if err != nil {
				t.Fatal(err)
			}
			var want, got map[string]any
			if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			for field, value := range want {
				if !reflect.DeepEqual(got[field], value) {
					t.Errorf("%s lost or changed: got %#v, want %#v", field, got[field], value)
				}
			}
		})
	}
}

func TestIntegrationResponsePreservesAbsentTest(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target any
	}{
		{"detail", new(IntegrationDetail)},
		{"create", new(CreateIntegrationResponse)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(`{"integration_id":42}`), tc.target); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(tc.target)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if _, ok := got["test"]; ok {
				t.Fatalf("absent test result became %s", encoded)
			}
		})
	}
}

func TestIntegrationRequestContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target any
		body   string
	}{
		{"create", new(CreateIntegrationRequest), `{"plugin_type":"standard.alert","integration_key":"0123456789abcdef0123456789abcdef100","is_test":true,"settings":{"headers":{}}}`},
		{"update", new(UpdateIntegrationRequest), `{"integration_id":42,"team_id":0,"is_test":true,"settings":{"headers":{}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(tc.body), tc.target); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(tc.target)
			if err != nil {
				t.Fatal(err)
			}
			var want, got map[string]any
			if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("request fields lost: got %s, want %s", encoded, tc.body)
			}
		})
	}
}
