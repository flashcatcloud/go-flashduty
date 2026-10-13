package flashduty

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAutomationRuleUpdateRequestTimezone(t *testing.T) {
	omitted, err := json.Marshal(&AutomationRuleUpdateRequest{RuleID: "arule_1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(omitted), "timezone") {
		t.Fatalf("nil timezone must be omitted, got %s", omitted)
	}

	empty, err := json.Marshal(&AutomationRuleUpdateRequest{
		RuleID:   "arule_1",
		Timezone: String(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(empty), `"timezone":""`) {
		t.Fatalf("empty timezone must be sent, got %s", empty)
	}

	named, err := json.Marshal(&AutomationRuleUpdateRequest{
		RuleID:   "arule_1",
		Timezone: String("Asia/Shanghai"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(named), `"timezone":"Asia/Shanghai"`) {
		t.Fatalf("named timezone must be sent, got %s", named)
	}
}
