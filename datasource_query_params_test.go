package flashduty

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDatasourceQueryInvokeRequestSLSWireFormat(t *testing.T) {
	// Beyond 2^53: must survive the wire as exact integers.
	from := int64(9007199254740993)
	to := int64(9007199254740994)
	params := &SLSQueryParams{
		Expr:     "status: 500",
		Project:  "my-project",
		Logstore: "my-logstore",
		PowerSQL: Bool(false),
		Limit:    Int64(50),
		Execution: DatasourceQueryExecution{
			Kind:   "window",
			FromMS: Int64(from),
			ToMS:   Int64(to),
		},
	}
	req, err := NewDatasourceQueryInvokeRequest(42, "sls.query", params)
	if err != nil {
		t.Fatal(err)
	}
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/monit/datasource/tools/invoke" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"powersql":false`) {
			t.Errorf("powersql lost or stringified: %s", body)
		}
		if !strings.Contains(string(body), `"from_ms":9007199254740993`) || !strings.Contains(string(body), `"to_ms":9007199254740994`) {
			t.Errorf("epoch millis lost precision or changed form: %s", body)
		}
		decoder := json.NewDecoder(strings.NewReader(string(body)))
		decoder.UseNumber()
		var input map[string]any
		if err := decoder.Decode(&input); err != nil {
			t.Fatal(err)
		}
		execution, ok := input["params"].(map[string]any)["execution"].(map[string]any)
		if !ok {
			t.Fatalf("execution missing: %s", body)
		}
		fromWire, ok := execution["from_ms"].(json.Number)
		if !ok {
			t.Fatalf("from_ms is not a JSON number: %s", body)
		}
		if fromWire.String() != "9007199254740993" {
			t.Errorf("from_ms precision lost: %s", fromWire)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"request_id":"trace-query","data":{"datasource_id":42,"tool":"sls.query","data":{"rows":[]}}}`)
	})
	result, _, err := client.DataSources.ToolsInvoke(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tool != "sls.query" || string(result.Data) != `{"rows":[]}` {
		t.Fatalf("response changed: %+v", result)
	}
}

func TestVictoriaLogsQueryParamsInstantStatsOmitsRawLogFields(t *testing.T) {
	params := &VictoriaLogsQueryParams{
		Expr: `* | stats count() as total`,
		Execution: DatasourceQueryExecution{
			Kind:   "instant",
			FromMS: Int64(1757433600000),
			ToMS:   Int64(1757520000000),
		},
	}
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"limit"`) || strings.Contains(string(raw), `"direction"`) {
		t.Fatalf("instant stats params carry raw-log keys: %s", raw)
	}
	if !strings.Contains(string(raw), `"from_ms":1757433600000`) {
		t.Fatalf("from_ms missing or not numeric: %s", raw)
	}
}

func TestNewDatasourceQueryInvokeRequestRejectsNilParams(t *testing.T) {
	req, err := NewDatasourceQueryInvokeRequest(42, "sls.query", nil)
	if err == nil || req != nil {
		t.Fatalf("nil params must fail: req=%+v err=%v", req, err)
	}
}
