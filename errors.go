package flashduty

import (
	"fmt"
	"net/http"
	"time"
)

// DutyError is the Flashduty API error object carried inside the response
// envelope's "error" field.
type DutyError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *DutyError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

func (e *DutyError) codeOr(d string) string {
	if e == nil {
		return d
	}
	return e.Code
}

func (e *DutyError) errMessageOr(d string) string {
	if e == nil || e.Message == "" {
		return d
	}
	return e.Message
}

// ErrorResponse is returned by any Flashduty API call that does not succeed —
// either the envelope carried an error or the HTTP status was non-2xx. Recover
// it with errors.As to inspect Code (see the generated ErrorCode constants) and
// RequestID.
type ErrorResponse struct {
	Response  *http.Response `json:"-"`
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id"`
}

func (e *ErrorResponse) Error() string {
	status := 0
	if e.Response != nil {
		status = e.Response.StatusCode
	}
	if e.Code != "" {
		return fmt.Sprintf("flashduty: %s (code %s, http %d, request_id %s)", e.Message, e.Code, status, e.RequestID)
	}
	return fmt.Sprintf("flashduty: %s (http %d, request_id %s)", e.Message, status, e.RequestID)
}

// RateLimitError is returned when the API responds 429. It embeds the standard
// *ErrorResponse (so errors.As(err, &target) for a *ErrorResponse still works)
// and adds the Retry-After hint. Inspect it with errors.As(err, &target) for a
// *RateLimitError.
type RateLimitError struct {
	*ErrorResponse
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s (retry after %s)", e.ErrorResponse.Error(), e.RetryAfter)
	}
	return e.ErrorResponse.Error()
}

// Unwrap lets errors.As reach the embedded *ErrorResponse.
func (e *RateLimitError) Unwrap() error { return e.ErrorResponse }
