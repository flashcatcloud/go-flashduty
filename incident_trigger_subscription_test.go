package flashduty

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIncidentTriggerSubscriptionUpsertRequestPreservesDisabledValue(t *testing.T) {
	var req IncidentTriggerSubscriptionUpsertRequest
	if err := json.Unmarshal([]byte(`{
		"source":"ai_sre_automation",
		"consumer":"fc_safari",
		"consumer_ref":"auto_1",
		"channel_ids":[2468013579],
		"severities":["Critical"],
		"enabled":false
	}`), &req); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(&req)
	if err != nil {
		t.Fatal(err)
	}
	got := string(payload)
	if !strings.Contains(got, `"enabled":false`) {
		t.Fatalf("payload %s missing explicit enabled=false", got)
	}
}
