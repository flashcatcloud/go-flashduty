package flashduty

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestSpecExamplesRoundTrip decodes every endpoint's canonical response example
// (vendored in the OpenAPI spec) into its generated Go data type via the same
// envelope→data path do() uses. A decode error means the generated type does not
// match the documented payload shape (e.g. a scalar typed wrong). This validates
// the whole generated type layer against real-shaped data without a live API.
func TestSpecExamplesRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("openapi/openapi.en.json")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	// Parse leniently: path-item values are method->operation, but a path item
	// may also carry a sibling "parameters" array — so read operations as raw.
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}

	tested := 0
	for path, methods := range spec.Paths {
		for method, opRaw := range methods {
			m := strings.ToUpper(method)
			if m != "GET" && m != "POST" {
				continue // skip "parameters" and other non-method keys
			}
			dec, ok := exampleDataDecoders[m+" "+path]
			if !ok {
				continue // endpoint returns no typed data
			}

			var op struct {
				Responses map[string]struct {
					Content map[string]struct {
						Example json.RawMessage `json:"example"`
					} `json:"content"`
				} `json:"responses"`
			}
			if err := json.Unmarshal(opRaw, &op); err != nil {
				t.Errorf("%s %s: parse operation: %v", m, path, err)
				continue
			}
			example := op.Responses["200"].Content["application/json"].Example
			if len(example) == 0 {
				continue
			}

			// Examples are full envelopes; decode data like do() does.
			var env struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(example, &env); err != nil {
				t.Errorf("%s %s: malformed example envelope: %v", m, path, err)
				continue
			}
			if len(env.Data) == 0 {
				continue
			}
			if err := dec(env.Data); err != nil {
				t.Errorf("%s %s: example data does not fit generated type: %v", m, path, err)
				continue
			}
			tested++
		}
	}

	// Guard against the test silently exercising nothing (e.g. spec path change).
	// ~135 endpoints return typed data with an example; require a healthy floor.
	if tested < 120 {
		t.Fatalf("expected to round-trip most endpoints, only exercised %d", tested)
	}
	t.Logf("round-tripped %d spec examples into generated types", tested)
}
