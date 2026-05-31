package flashduty

// Pointer-helper constructors, in the style of go-github's Ptr helpers.
//
// Optional request fields that the backend models as tri-state (a pointer, so
// it can distinguish "unset" from "the zero value") are generated as Go
// pointers — e.g. AlertListRequest.IsActive is *bool. Wrap a literal with these
// helpers to set such a field, including to its zero value:
//
//	req := &flashduty.AlertListRequest{IsActive: flashduty.Bool(false)} // show resolved
//
// A nil pointer omits the field entirely (no filter / no change).

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }

// Int returns a pointer to v.
func Int(v int) *int { return &v }

// Int64 returns a pointer to v.
func Int64(v int64) *int64 { return &v }

// Uint64 returns a pointer to v.
func Uint64(v uint64) *uint64 { return &v }

// Float64 returns a pointer to v.
func Float64(v float64) *float64 { return &v }

// String returns a pointer to v.
func String(v string) *string { return &v }
