package flashduty

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestListOptionsJSONTags(t *testing.T) {
	b, _ := json.Marshal(ListOptions{Page: 2, Limit: 50, SearchAfterCtx: "c"})
	got := string(b)
	for _, want := range []string{`"p":2`, `"limit":50`, `"search_after_ctx":"c"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("ListOptions JSON = %s, missing %s", got, want)
		}
	}
	// Zero values must be omitted so they never override server defaults.
	if z, _ := json.Marshal(ListOptions{}); string(z) != "{}" {
		t.Fatalf("empty ListOptions = %s, want {}", z)
	}
}
