package flashduty

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorCodeOf(t *testing.T) {
	notFound := &ErrorResponse{Code: string(ErrorCodeResourceNotFound)}

	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "plain ErrorResponse",
			err:  notFound,
			want: ErrorCodeResourceNotFound,
		},
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "non-API error",
			err:  errors.New("boom"),
			want: "",
		},
		{
			name: "rate limit error unwraps to ErrorResponse",
			err:  &RateLimitError{ErrorResponse: &ErrorResponse{Code: string(ErrorCodeRequestTooFrequently)}},
			want: ErrorCodeRequestTooFrequently,
		},
		{
			name: "wrapped via fmt.Errorf",
			err:  fmt.Errorf("ctx: %w", notFound),
			want: ErrorCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorCodeOf(tt.err); got != tt.want {
				t.Errorf("ErrorCodeOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "matching code",
			err:  &ErrorResponse{Code: string(ErrorCodeResourceNotFound)},
			want: true,
		},
		{
			name: "different code",
			err:  &ErrorResponse{Code: string(ErrorCodeUnauthorized)},
			want: false,
		},
		{
			name: "non-API error",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "RateLimitError with empty code",
			err:  &RateLimitError{ErrorResponse: &ErrorResponse{}},
			want: true,
		},
		{
			name: "RateLimitError nil embedded code",
			err:  &RateLimitError{ErrorResponse: &ErrorResponse{Code: string(ErrorCodeRequestTooFrequently)}},
			want: true,
		},
		{
			name: "plain RequestTooFrequently code",
			err:  &ErrorResponse{Code: string(ErrorCodeRequestTooFrequently)},
			want: true,
		},
		{
			name: "plain not-found",
			err:  &ErrorResponse{Code: string(ErrorCodeResourceNotFound)},
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimited(tt.err); got != tt.want {
				t.Errorf("IsRateLimited() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "matching code",
			err:  &ErrorResponse{Code: string(ErrorCodeUnauthorized)},
			want: true,
		},
		{
			name: "different code",
			err:  &ErrorResponse{Code: string(ErrorCodeAccessDenied)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnauthorized(tt.err); got != tt.want {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAccessDenied(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "matching code",
			err:  &ErrorResponse{Code: string(ErrorCodeAccessDenied)},
			want: true,
		},
		{
			name: "different code",
			err:  &ErrorResponse{Code: string(ErrorCodeUnauthorized)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAccessDenied(tt.err); got != tt.want {
				t.Errorf("IsAccessDenied() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInvalidParameter(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "matching code",
			err:  &ErrorResponse{Code: string(ErrorCodeInvalidParameter)},
			want: true,
		},
		{
			name: "different code",
			err:  &ErrorResponse{Code: string(ErrorCodeBadRequest)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInvalidParameter(tt.err); got != tt.want {
				t.Errorf("IsInvalidParameter() = %v, want %v", got, tt.want)
			}
		})
	}
}
