package flashduty

import (
	"net/url"
	"strings"
	"testing"
)

func TestSanitizeURLRedactsAppKey(t *testing.T) {
	u, _ := url.Parse("https://api.flashcat.cloud/incident/list?app_key=SECRET&p=1")
	got := sanitizeURL(u)
	if strings.Contains(got, "SECRET") {
		t.Fatalf("app_key leaked: %s", got)
	}
	// url.Values.Encode percent-encodes the brackets, so the marker appears as
	// %5BREDACTED%5D; assert on the stable "REDACTED" token.
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("expected redaction marker, got %s", got)
	}
}

func TestSanitizeBodyRedactsSecrets(t *testing.T) {
	in := `{"name":"n","password":"hunter2","env":{"OPENAI_API_KEY":"sk-xyz"}}`
	got := sanitizeBody(in)
	for _, leak := range []string{"hunter2", "sk-xyz"} {
		if strings.Contains(got, leak) {
			t.Fatalf("secret leaked: %s", got)
		}
	}
	if !strings.Contains(got, `"name"`) {
		t.Fatalf("non-secret field dropped: %s", got)
	}
}

func TestSanitizeBodyDoesNotLogUnstructuredContent(t *testing.T) {
	marker := strings.Repeat("sensitive-", 2)
	got := sanitizeBody(`<html>request /incident/list?app_key=` + marker + ` timed out</html>`)
	if got != "[NON_JSON_BODY]" {
		t.Fatalf("sanitizeBody(non-JSON) = %q, want fixed marker", got)
	}
}
