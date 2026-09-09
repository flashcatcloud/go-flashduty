package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAlertRuleUpdateV2PreservesInvestigationTargetsPresence(t *testing.T) {
	tests := []struct {
		name    string
		targets []InvestigationTarget
		want    string
	}{
		{name: "omit keeps existing targets"},
		{name: "empty clears targets", targets: []InvestigationTarget{}, want: `[]`},
		{
			name: "replace targets",
			targets: []InvestigationTarget{{
				Kind: "dashboard",
				Dashboard: DashboardInvestigationTarget{
					DashboardID: "01900000-0000-7000-8000-000000000001",
				},
			}},
			want: `[{"dashboard":{"dashboard_id":"01900000-0000-7000-8000-000000000001"},"kind":"dashboard"}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/monit/rule/v2/update" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode request: %v", err)
				}
				if got := string(body["investigation_targets"]); got != tt.want {
					t.Errorf("investigation_targets = %q, want %q", got, tt.want)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"data": body})
			})
			_, _, err := client.AlertRules.WriteUpdateV2(context.Background(), &AlertRuleV2{
				ID:                   123,
				InvestigationTargets: tt.targets,
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
