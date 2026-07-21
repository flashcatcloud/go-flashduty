package flashduty

import (
	"encoding/json"
	"testing"
)

func TestDiagnoseResponseOmitsAbsentUnionFields(t *testing.T) {
	const metricResponse = `{
		"schema_version":"2",
		"operation":"metric_trends",
		"ds_type":"prometheus",
		"ds_name":"prod-prometheus",
		"query":"up",
		"window":{"start":"2026-07-14T06:00:00Z","end":"2026-07-14T07:00:00Z"},
		"results":[{
			"method":"window_compare",
			"window":{"start":"2026-07-14T06:00:00Z","end":"2026-07-14T07:00:00Z"},
			"summary":{"series_total":1,"series_analyzed":1,"selected_series_total":0,"series_returned":0,"analysis_truncated":false,"evidence_summary":"No series matched the selection rules."},
			"series_evidence":[],
			"warnings":[]
		}]
	}`

	var response DiagnoseResponse
	if err := json.Unmarshal([]byte(metricResponse), &response); err != nil {
		t.Fatalf("unmarshal metric diagnose response: %v", err)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal metric diagnose response: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("decode marshalled metric response: %v", err)
	}
	if _, found := output["data_handling"]; found {
		t.Fatalf("metric result fabricated data_handling: %s", encoded)
	}
	result := output["results"].([]any)[0].(map[string]any)
	if _, found := result["pattern_evidence"]; found {
		t.Fatalf("metric result fabricated pattern_evidence: %s", encoded)
	}
	if _, found := result["series_evidence"]; !found {
		t.Fatalf("metric result lost series_evidence: %s", encoded)
	}
	summary := result["summary"].(map[string]any)
	if _, found := summary["current_sample"]; found {
		t.Fatalf("metric result fabricated log summary: %s", encoded)
	}
	if _, found := summary["series_total"]; !found {
		t.Fatalf("metric result lost metric summary: %s", encoded)
	}
}

func TestDiagnoseResponseOmitsAbsentMetricEvidenceFields(t *testing.T) {
	const metricResponse = `{
		"schema_version":"2",
		"operation":"metric_trends",
		"ds_type":"prometheus",
		"ds_name":"prod-prometheus",
		"query":"up",
		"window":{"start":"2026-07-14T06:00:00Z","end":"2026-07-14T07:00:00Z"},
		"results":[{
			"method":"window_compare",
			"window":{"start":"2026-07-14T06:00:00Z","end":"2026-07-14T07:00:00Z"},
			"summary":{"series_total":1,"series_analyzed":1,"selected_series_total":1,"series_returned":1,"analysis_truncated":false,"evidence_summary":"One series changed."},
			"series_evidence":[{"labels":{"instance":"api-1"},"observations":["The current average increased."]}],
			"warnings":[]
		}]
	}`

	var response DiagnoseResponse
	if err := json.Unmarshal([]byte(metricResponse), &response); err != nil {
		t.Fatalf("unmarshal metric diagnose response: %v", err)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal metric diagnose response: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("decode marshalled metric response: %v", err)
	}
	evidence := output["results"].([]any)[0].(map[string]any)["series_evidence"].([]any)[0].(map[string]any)
	for _, field := range []string{"comparison_status", "current_window_stats", "baseline_window_stats"} {
		if _, found := evidence[field]; found {
			t.Fatalf("metric evidence fabricated %s: %s", field, encoded)
		}
	}
}

func TestDiagnoseResponseOmitsAbsentLogEvidenceFields(t *testing.T) {
	const logResponse = `{
		"schema_version":"2",
		"operation":"log_patterns",
		"ds_type":"elasticsearch",
		"ds_name":"prod-logs",
		"query":"service:api",
		"window":{"start":"2026-07-14T06:00:00Z","end":"2026-07-14T07:00:00Z"},
		"results":[{
			"method":"pattern_snapshot",
			"window":{"start":"2026-07-14T06:00:00Z","end":"2026-07-14T07:00:00Z"},
			"summary":{
				"current_sample":{"logs_scanned":10,"patterns_aggregated":1,"logs_not_aggregated_due_to_cluster_limit":0,"pattern_matching_limited":false,"truncated":false},
				"aggregated_pattern_evidence_total":1,
				"pattern_evidence_returned":1,
				"pattern_evidence_truncated_by_max_patterns":false,
				"evidence_summary":"One pattern was observed."
			},
			"pattern_evidence":[{"pattern_id":"abc123","pattern_template":"request failed"}],
			"warnings":[]
		}],
		"data_handling":{"log_redaction_applied":true,"log_redaction_coverage":"best_effort","untrusted_data_fields":[]}
	}`

	var response DiagnoseResponse
	if err := json.Unmarshal([]byte(logResponse), &response); err != nil {
		t.Fatalf("unmarshal log diagnose response: %v", err)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal log diagnose response: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("decode marshalled log response: %v", err)
	}
	result := output["results"].([]any)[0].(map[string]any)
	evidence := result["pattern_evidence"].([]any)[0].(map[string]any)
	for _, field := range []string{"comparison_status", "current_window", "baseline_window", "observations", "redacted_log_examples"} {
		if _, found := evidence[field]; found {
			t.Fatalf("log evidence fabricated %s: %s", field, encoded)
		}
	}
	summary := result["summary"].(map[string]any)
	for _, field := range []string{"baseline_sample", "patterns_aggregated_only_in_baseline_sample"} {
		if _, found := summary[field]; found {
			t.Fatalf("log summary fabricated %s: %s", field, encoded)
		}
	}
	currentSample := summary["current_sample"].(map[string]any)
	if _, found := currentSample["sampling_bias"]; found {
		t.Fatalf("log sample fabricated sampling_bias: %s", encoded)
	}
}
