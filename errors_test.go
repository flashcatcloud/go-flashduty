package flashduty

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestErrorResponseFormatAndUnwrap(t *testing.T) {
	apiErr := &ErrorResponse{
		Response:  &http.Response{StatusCode: 404},
		Code:      "incident_not_found",
		Message:   "incident not found",
		RequestID: "REQ123",
	}
	var err error = apiErr
	got := err.Error()
	for _, want := range []string{"incident not found", "incident_not_found", "404", "REQ123"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Error() = %q, missing %q", got, want)
		}
	}
	var target *ErrorResponse
	if !errors.As(err, &target) || target.Code != "incident_not_found" {
		t.Fatalf("errors.As failed to recover *ErrorResponse")
	}
}
