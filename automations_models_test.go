package flashduty

import (
	"encoding/json"
	"reflect"
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

func TestAutomationRuleCreateRequestExecutionDestination(t *testing.T) {
	omitted, err := json.Marshal(&AutomationRuleCreateRequest{
		Name:     "Daily SRE brief",
		CronExpr: "0 9 * * *",
		Prompt:   "Summarize yesterday's incidents",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"run_in", "team_id"} {
		if strings.Contains(string(omitted), `"`+field+`"`) {
			t.Fatalf("omitted %s must stay off the wire, got %s", field, omitted)
		}
	}

	explicit, err := json.Marshal(&AutomationRuleCreateRequest{
		Name:     "Daily SRE brief",
		CronExpr: "0 9 * * *",
		Prompt:   "Summarize yesterday's incidents",
		RunIn:    String("current-chat"),
		TeamID:   Int64(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"run_in":"current-chat"`, `"team_id":0`} {
		if !strings.Contains(string(explicit), want) {
			t.Fatalf("explicit destination must send %s, got %s", want, explicit)
		}
	}
}

func TestAutomationRuleUpdateRequestHasNoExecutionTarget(t *testing.T) {
	typ := reflect.TypeFor[AutomationRuleUpdateRequest]()
	for _, field := range []string{"TeamID", "RunIn"} {
		if _, ok := typ.FieldByName(field); ok {
			t.Fatalf("update must not expose immutable execution target field %s", field)
		}
	}
}
