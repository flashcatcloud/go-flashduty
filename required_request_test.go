package flashduty

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestResetPostMortemContentRequestMarshalsZeroExpectedRevision(t *testing.T) {
	raw, err := json.Marshal(ResetPostMortemContentRequest{
		PostMortemID:     "post-mortem-id",
		Markdown:         "# Report",
		ExpectedRevision: 0,
		IdempotencyKey:   "idempotency-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"expected_revision":0`)) {
		t.Fatalf("required expected_revision missing from JSON: %s", raw)
	}
}
