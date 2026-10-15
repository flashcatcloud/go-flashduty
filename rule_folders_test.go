package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestRuleFoldersAppKeyRequests(t *testing.T) {
	for _, action := range []string{"list", "health", "create", "update", "move", "delete"} {
		t.Run(action, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/monit/folder/"+action || r.URL.Query().Get("app_key") != "KEY" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Redacted())
				}
				if action == "move" {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					// Moving to root requires an explicit zero on the wire.
					if !reflect.DeepEqual(body, map[string]any{"id": float64(12), "dest_parent_id": float64(0)}) {
						t.Errorf("move body = %#v", body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				data := any(map[string]any{})
				switch action {
				case "list":
					data = []map[string]any{{"id": 12, "name": "Checkout"}}
				case "health":
					data = map[string]any{"items": []map[string]any{{"folder_id": 12, "has_issue": true}}}
				case "create", "update":
					data = map[string]any{"id": 12, "name": "Checkout"}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
			})
			ctx := context.Background()
			var err error
			switch action {
			case "list":
				out, _, e := client.RuleFolders.List(ctx)
				err = e
				if e == nil && (len(*out) != 1 || (*out)[0].ID != 12) {
					t.Errorf("list = %#v", out)
				}
			case "health":
				out, _, e := client.RuleFolders.ReadHealth(ctx)
				err = e
				if e == nil && (len(out.Items) != 1 || out.Items[0].FolderID != 12 || !out.Items[0].HasIssue) {
					t.Errorf("health = %#v", out)
				}
			case "create":
				out, _, e := client.RuleFolders.WriteCreate(ctx, &FolderCreateRequest{Name: "Checkout"})
				err = e
				if e == nil && out.ID != 12 {
					t.Errorf("create = %#v", out)
				}
			case "update":
				out, _, e := client.RuleFolders.WriteUpdate(ctx, &FolderUpdateRequest{ID: 12, Name: "Checkout"})
				err = e
				if e == nil && out.ID != 12 {
					t.Errorf("update = %#v", out)
				}
			case "move":
				_, err = client.RuleFolders.WriteMove(ctx, &FolderMoveRequest{ID: 12, DestParentID: 0})
			case "delete":
				_, err = client.RuleFolders.WriteDelete(ctx, &FolderDeleteRequest{ID: 12})
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
