package main

import (
	"strings"
	"testing"
)

// newTestGen returns a Gen with every map field initialized, mirroring run()'s
// construction, so g.queue (invoked while resolving inline nullable objects)
// and g.isStructSchema (which walks g.schemas for $ref targets) don't panic
// on a nil map write/read.
func newTestGen(schemas map[string]any) *Gen {
	return &Gen{
		schemas:    schemas,
		emptyTypes: map[string]bool{},
		queued:     map[string]bool{},
		synth:      map[string]any{},
		reqGoNames: map[string]bool{},
		reqSynth:   map[string]bool{},
	}
}

// TestEmitStructPointerWrapsNullableRefField is the load-bearing proof for the
// nullable-$ref-field fix: a response-side struct field whose schema is a
// `$ref` to an object type, marked `nullable: true` (the only spelling this
// org's OpenAPI can give a struct field — see isNullable's doc comment),
// must generate as a pointer. Before the fix, isNullable ignored the boolean
// `nullable` marker entirely and goTypeOf's $ref branch never consulted
// nullability at all, so this field generated as a bare value type: decoding
// a real `null` from the API silently produced a zero-value TargetGroup{},
// indistinguishable from a legitimate all-zero-fields object.
func TestEmitStructPointerWrapsNullableRefField(t *testing.T) {
	g := newTestGen(map[string]any{
		"TargetGroup": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kind": map[string]any{"type": "string"},
			},
		},
	})

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target": map[string]any{
				"$ref":        "#/components/schemas/TargetGroup",
				"nullable":    true,
				"description": "Resolved target. Null when it could not be resolved.",
			},
			"fallback_target": map[string]any{
				"$ref": "#/components/schemas/TargetGroup",
			},
		},
	}

	src := g.emitStruct("ResponseLike", schema)

	if !strings.Contains(src, "Target *TargetGroup") {
		t.Fatalf("nullable $ref field did not pointer-wrap; got:\n%s", src)
	}
	if !strings.Contains(src, "FallbackTarget TargetGroup") || strings.Contains(src, "FallbackTarget *TargetGroup") {
		t.Fatalf("non-nullable $ref field was unexpectedly pointer-wrapped (or missing); got:\n%s", src)
	}
}

// TestIsNullableRecognizesBothMarkerForms locks in the two nullability
// spellings this org's OpenAPI uses: the 3.1 union `type: ["T", "null"]`
// (every nullable scalar field) and the 3.0 boolean `nullable: true` (used
// for object fields, since $ref/inline-object siblings can't carry a type
// array with a meaningful non-null member naming the $ref). Before the fix,
// only the union form was recognized.
func TestIsNullableRecognizesBothMarkerForms(t *testing.T) {
	cases := map[string]map[string]any{
		"union form (scalar)":   {"type": []any{"integer", "null"}},
		"boolean form (object)": {"type": "object", "nullable": true},
	}
	for name, s := range cases {
		if !isNullable(s) {
			t.Errorf("%s: isNullable = false, want true", name)
		}
	}
	if isNullable(map[string]any{"type": "object"}) {
		t.Error("plain object schema: isNullable = true, want false")
	}
}

// TestEmitStructInlineNullableObjectPointerWraps exercises the other spelling
// a nullable struct field can take: an inline (non-$ref) object schema using
// the 3.1 `type: ["object", "null"]` union form. A struct-shaped inline
// field using this form must pointer-wrap exactly like the $ref case.
func TestEmitStructInlineNullableObjectPointerWraps(t *testing.T) {
	g := newTestGen(map[string]any{})

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target": map[string]any{
				"type": []any{"object", "null"},
				"properties": map[string]any{
					"kind": map[string]any{"type": "string"},
				},
			},
		},
	}

	src := g.emitStruct("ToolResponseLike", schema)
	if !strings.Contains(src, "Target *ToolResponseLikeTarget") {
		t.Fatalf("inline nullable object field did not pointer-wrap; got:\n%s", src)
	}
}

// TestEmitStructNullableScalarResponseFieldStaysValue guards the deliberate,
// unchanged policy for nullable *scalar* response fields (see the comment
// above the inReq-gated pointerization in emitStruct): a nullable scalar on
// the response side stays a bare value type — decoding null yields the zero
// value, which the team accepted as an acceptable ambiguity for scalars only.
// The struct fix must not broaden that policy.
func TestEmitStructNullableScalarResponseFieldStaysValue(t *testing.T) {
	g := newTestGen(map[string]any{})

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{"type": []any{"integer", "null"}},
		},
	}

	src := g.emitStruct("ScheduleItemLike", schema)
	if !strings.Contains(src, "Status int64") || strings.Contains(src, "Status *int64") {
		t.Fatalf("nullable scalar response field changed from the documented value-type policy; got:\n%s", src)
	}
}

func TestEmitStructNullableOptionalResponseFieldUsesSinglePointer(t *testing.T) {
	g := newTestGen(map[string]any{
		"TargetGroup": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kind": map[string]any{"type": "string"},
			},
		},
	})

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target": map[string]any{
				"$ref":                         "#/components/schemas/TargetGroup",
				"nullable":                     true,
				"x-flashduty-preserve-absence": true,
			},
		},
	}

	src := g.emitStruct("ResponseLike", schema)
	if !strings.Contains(src, "Target *TargetGroup") || strings.Contains(src, "Target **TargetGroup") {
		t.Fatalf("nullable optional response field must use exactly one pointer; got:\n%s", src)
	}
}

func TestEmitStructRequiredRequestScalarDoesNotOmitZero(t *testing.T) {
	g := newTestGen(map[string]any{})
	g.reqGoNames["ResetRequest"] = true

	schema := map[string]any{
		"type":     "object",
		"required": []any{"expected_revision"},
		"properties": map[string]any{
			"expected_revision": map[string]any{
				"type":    "integer",
				"minimum": 0,
			},
		},
	}

	src := g.emitStruct("ResetRequest", schema)
	if !strings.Contains(src, `ExpectedRevision int64 `+"`"+`json:"expected_revision" toon:"expected_revision"`+"`") {
		t.Fatalf("required request scalar must keep its zero value on the wire; got:\n%s", src)
	}
}

func TestEmitStructRequiredNullableRequestScalarOmitsNil(t *testing.T) {
	g := newTestGen(map[string]any{})
	g.reqGoNames["ResetRequest"] = true

	schema := map[string]any{
		"type":     "object",
		"required": []any{"expected_revision"},
		"properties": map[string]any{
			"expected_revision": map[string]any{
				"type":    []any{"integer", "null"},
				"minimum": 0,
			},
		},
	}

	src := g.emitStruct("ResetRequest", schema)
	if !strings.Contains(src, `ExpectedRevision *int64 `+"`"+`json:"expected_revision,omitempty" toon:"expected_revision,omitempty"`+"`") {
		t.Fatalf("required nullable request scalar must distinguish nil from a present zero; got:\n%s", src)
	}
}

func TestMergeAllOfKeepsRequiredRequestFields(t *testing.T) {
	g := newTestGen(map[string]any{
		"BaseRequest": map[string]any{
			"type":     "object",
			"required": []any{"base_count"},
			"properties": map[string]any{
				"base_count": map[string]any{"type": "integer"},
			},
		},
	})
	g.reqGoNames["CombinedRequest"] = true

	merged := g.mergeAllOf(map[string]any{
		"allOf": []any{
			map[string]any{"$ref": "#/components/schemas/BaseRequest"},
			map[string]any{
				"type":     "object",
				"required": []any{"label"},
				"properties": map[string]any{
					"label": map[string]any{"type": "string"},
				},
			},
		},
	})

	src := g.emitStruct("CombinedRequest", merged)
	for _, want := range []string{
		`BaseCount int64 ` + "`" + `json:"base_count" toon:"base_count"` + "`",
		`Label string ` + "`" + `json:"label" toon:"label"` + "`",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("allOf required request field missing its wire requirement %q; got:\n%s", want, src)
		}
	}
}

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
