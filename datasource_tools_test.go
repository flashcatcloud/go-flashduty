package flashduty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDatasourceToolsInvokePreservesJSON(t *testing.T) {
	params := json.RawMessage(`{"cursor":9007199254740993,"threshold":1.000000000000000001,"nested":{"empty":[],"flag":false}}`)
	evidence := `{"counter":18446744073709551615,"ratio":0.1234567890123456789,"values":[null,false,0]}`
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/monit/datasource/tools/invoke" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var input DatasourceToolInvokeRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input.DatasourceID != 42 || input.Tool != "mysql.overview" || string(input.Params) != string(params) {
			t.Errorf("request changed: %+v", input)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"request_id":"trace-tools","data":{"datasource_id":42,"tool":"mysql.overview","data":%s,"summary":"bounded evidence","truncated":{"reason":"row_limit"}}}`, evidence)
	})
	result, resp, err := client.DataSources.ToolsInvoke(context.Background(), &DatasourceToolInvokeRequest{DatasourceID: 42, Tool: "mysql.overview", Params: params})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RequestID != "trace-tools" || result.DatasourceID != 42 || result.Tool != "mysql.overview" || string(result.Data) != evidence || result.Summary == nil || result.Truncated == nil || result.Truncated.Reason != "row_limit" {
		t.Fatalf("response changed: %+v", result)
	}
	raw, err := json.Marshal(result)
	if err != nil || !strings.Contains(string(raw), evidence) {
		t.Fatalf("round trip changed evidence: %s, %v", raw, err)
	}
}

func TestDatasourceToolsInvokeHTTPErrorReason(t *testing.T) {
	for _, tc := range []struct {
		status int
		reason string
	}{{400, "tool_not_supported"}, {409, "datasource_disabled"}, {429, "overloaded"}, {503, "edge_upgrade_required"}, {504, "timeout"}} {
		t.Run(tc.reason, func(t *testing.T) {
			calls := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				fmt.Fprintf(w, `{"request_id":"trace-error","error":{"code":"InvalidParameter","reason":%q,"message":"tool unavailable"}}`, tc.reason)
			})
			result, _, err := client.DataSources.ToolsInvoke(context.Background(), &DatasourceToolInvokeRequest{DatasourceID: 42, Tool: "mysql.overview"})
			var apiErr *ErrorResponse
			if result != nil || !errors.As(err, &apiErr) || apiErr.Reason != tc.reason || apiErr.Response.StatusCode != tc.status || apiErr.RequestID != "trace-error" || calls != 1 {
				t.Fatalf("error lost or call replayed: result=%+v error=%+v calls=%d", result, err, calls)
			}
			if !strings.Contains(err.Error(), "reason "+tc.reason) || !strings.Contains(err.Error(), "request_id trace-error") {
				t.Fatalf("CLI error text lost reason or request ID: %s", err)
			}
		})
	}
}

func TestDatasourceWritePresenceAndDiagnosticSecrets(t *testing.T) {
	for _, action := range []string{"create", "update"} {
		for _, present := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/present=%v", action, present), func(t *testing.T) {
				client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/monit/datasource/"+action {
						t.Errorf("wrong path: %s", r.URL.Path)
					}
					body, _ := io.ReadAll(r.Body)
					var input map[string]json.RawMessage
					if err := json.Unmarshal(body, &input); err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{"enabled", "alerting_enabled"} {
						value, ok := input[key]
						if ok != present || (present && string(value) != "false") {
							t.Errorf("%s presence lost: %s", key, body)
						}
					}
					var payload map[string]map[string]json.RawMessage
					if err := json.Unmarshal(input["payload"], &payload); err != nil {
						t.Fatal(err)
					}
					password, ok := payload["redis_node"]["password"]
					if ok != present || (present && string(password) != `""`) {
						t.Errorf("secret presence lost: %s", body)
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"request_id":"write","data":{"id":42,"enabled":false,"alerting_enabled":false,"type_ident":"redis_node","payload":{"redis_node":{"database":0}}}}`)
				})
				var req DataSourceUpsertRequest
				raw := `{"id":42,"name":"cache","type_ident":"redis_node","edge_cluster_name":"default","address":"redis.example.com:6379","payload":{"redis_node":{"database":0}}}`
				if present {
					raw = `{"id":42,"name":"cache","type_ident":"redis_node","edge_cluster_name":"default","address":"redis.example.com:6379","enabled":false,"alerting_enabled":false,"payload":{"redis_node":{"database":0,"password":""}}}`
				}
				if err := json.Unmarshal([]byte(raw), &req); err != nil {
					t.Fatal(err)
				}
				var result *DataSourceItem
				var err error
				if action == "create" {
					result, _, err = client.DataSources.WriteCreate(context.Background(), &req)
				} else {
					result, _, err = client.DataSources.WriteUpdate(context.Background(), &req)
				}
				if err != nil {
					t.Fatal(err)
				}
				encoded, _ := json.Marshal(result)
				if !strings.Contains(string(encoded), `"enabled":false`) || !strings.Contains(string(encoded), `"alerting_enabled":false`) {
					t.Fatalf("false response fields lost: %s", encoded)
				}
			})
		}
	}
}

func TestErrorResponseReasonText(t *testing.T) {
	for _, tc := range []struct{ code, reason, want string }{
		{"InvalidParameter", "", "flashduty: unavailable (code InvalidParameter, http 503, request_id trace-error)"},
		{"", "", "flashduty: unavailable (http 503, request_id trace-error)"},
		{"", "edge_upgrade_required", "flashduty: unavailable (http 503, reason edge_upgrade_required, request_id trace-error)"},
	} {
		err := &ErrorResponse{Response: &http.Response{StatusCode: 503}, Code: tc.code, Reason: tc.reason, Message: "unavailable", RequestID: "trace-error"}
		if got := err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
}
