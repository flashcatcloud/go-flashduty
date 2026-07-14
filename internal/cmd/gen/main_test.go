package main

import "testing"

func TestMergeAllOfMergesObjectOneOfBranches(t *testing.T) {
	g := &Gen{schemas: map[string]any{
		"LogResult": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern_evidence": map[string]any{"type": "array"},
			},
		},
		"MetricResult": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"series_evidence": map[string]any{"type": "array"},
			},
		},
	}}

	merged := g.mergeAllOf(map[string]any{
		"oneOf": []any{
			map[string]any{"$ref": "#/components/schemas/LogResult"},
			map[string]any{"$ref": "#/components/schemas/MetricResult"},
		},
	})
	properties := asMap(merged["properties"])

	if properties["pattern_evidence"] == nil {
		t.Fatal("oneOf log branch field was not merged")
	}
	if properties["series_evidence"] == nil {
		t.Fatal("oneOf metric branch field was not merged")
	}
}

func TestMergeAllOfMarksOptionalOneOfProperties(t *testing.T) {
	g := &Gen{schemas: map[string]any{
		"LogResult": map[string]any{
			"type":     "object",
			"required": []any{"method", "pattern_evidence"},
			"properties": map[string]any{
				"method":           map[string]any{"type": "string"},
				"pattern_evidence": map[string]any{"type": "array"},
			},
		},
		"MetricResult": map[string]any{
			"type":     "object",
			"required": []any{"method", "series_evidence"},
			"properties": map[string]any{
				"method":          map[string]any{"type": "string"},
				"series_evidence": map[string]any{"type": "array"},
			},
		},
	}}

	merged := g.mergeAllOf(map[string]any{
		"oneOf": []any{
			map[string]any{"$ref": "#/components/schemas/LogResult"},
			map[string]any{"$ref": "#/components/schemas/MetricResult"},
		},
	})

	if optional, _ := merged["x-generator-optional-response"].(bool); !optional {
		t.Fatal("oneOf object merge did not preserve optional-response metadata")
	}
	if got := asSlice(merged["required"]); len(got) != 1 || got[0] != "method" {
		t.Fatalf("required oneOf intersection = %#v, want [method]", got)
	}
}
