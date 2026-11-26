package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

// The update endpoint keeps a field whose key is absent and clears one sent
// empty, so the request must carry exactly the keys the caller set.
func TestChangesUpdateSendsOnlySetKeys(t *testing.T) {
	var got map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/change/update" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"change_id": "664a1b2c3d4e5f6a7b8c9d0e", "change_key": "CHG-20261126-7Q2K", "change_status": "Processing",
		}})
	})

	out, _, err := client.Changes.Update(context.Background(), &UpdateChangeRequest{
		ChangeID:       "664a1b2c3d4e5f6a7b8c9d0e",
		ChangeStatus:   "Processing",
		Description:    String(""),
		PlannedEndTime: Int64(0),
		Checklist:      []ChangeChecklistItem{},
		Fields:         map[string]any{"risk_level": nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"change_id":        "664a1b2c3d4e5f6a7b8c9d0e",
		"change_status":    "Processing",
		"description":      "",
		"planned_end_time": float64(0),
		"checklist":        []any{},
		"fields":           map[string]any{"risk_level": nil},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body = %#v\nwant   %#v", got, want)
	}
	if out.ChangeStatus != "Processing" || out.ChangeKey != "CHG-20261126-7Q2K" {
		t.Errorf("out = %#v", out)
	}
}
