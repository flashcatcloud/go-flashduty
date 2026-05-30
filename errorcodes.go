package flashduty

import "errors"

// Error-code helpers.
//
// These predicates let callers branch on the server's machine-readable
// ErrorCode without string-comparing it themselves. Every helper unwraps the
// supplied error via errors.As, so they see through wrapped errors (e.g. an
// error produced by fmt.Errorf("...: %w", apiErr)) and through a
// *RateLimitError, which unwraps to its embedded *ErrorResponse.

// ErrorCodeOf extracts the API ErrorCode carried by err, if any.
//
// It searches the error chain with errors.As for an *ErrorResponse and returns
// ErrorCode(apiErr.Code). Because *RateLimitError unwraps to *ErrorResponse,
// this also resolves the code for rate-limit errors. If no *ErrorResponse is
// present in the chain, it returns the empty ErrorCode ("").
func ErrorCodeOf(err error) ErrorCode {
	var apiErr *ErrorResponse
	if errors.As(err, &apiErr) {
		return ErrorCode(apiErr.Code)
	}
	return ""
}

// IsNotFound reports whether err carries the ResourceNotFound API error code.
func IsNotFound(err error) bool {
	return ErrorCodeOf(err) == ErrorCodeResourceNotFound
}

// IsUnauthorized reports whether err carries the Unauthorized API error code.
func IsUnauthorized(err error) bool {
	return ErrorCodeOf(err) == ErrorCodeUnauthorized
}

// IsAccessDenied reports whether err carries the AccessDenied API error code.
func IsAccessDenied(err error) bool {
	return ErrorCodeOf(err) == ErrorCodeAccessDenied
}

// IsInvalidParameter reports whether err carries the InvalidParameter API error code.
func IsInvalidParameter(err error) bool {
	return ErrorCodeOf(err) == ErrorCodeInvalidParameter
}

// IsRateLimited reports whether err represents a rate-limit condition.
//
// It returns true when the error chain contains a *RateLimitError (regardless
// of the code it carries) or when the resolved code is RequestTooFrequently.
func IsRateLimited(err error) bool {
	var rl *RateLimitError
	if errors.As(err, &rl) {
		return true
	}
	return ErrorCodeOf(err) == ErrorCodeRequestTooFrequently
}
