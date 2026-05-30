package flashduty

import (
	"bytes"
	"strconv"
	"time"
)

// Timestamp is a Unix-seconds instant as it appears on the Flashduty API wire.
//
// It marshals to an RFC3339 string in the local timezone, so structured output
// is human- and LLM-readable instead of an opaque integer. It unmarshals from
// either a numeric epoch (the wire form) or an RFC3339 string (so a marshaled
// value round-trips). The zero value marshals to 0 — an unset sentinel, never a
// 1970 date — and is dropped by `json:",omitempty"`.
//
// Use Timestamp only for absolute instants. Durations, cyclic-window offsets,
// and counts stay int64.
type Timestamp int64

// Time returns the instant as a time.Time.
func (t Timestamp) Time() time.Time { return time.Unix(int64(t), 0) }

// Unix returns the raw wire value (Unix seconds).
func (t Timestamp) Unix() int64 { return int64(t) }

// IsZero reports whether the value is the unset sentinel (0).
func (t Timestamp) IsZero() bool { return t == 0 }

// String renders the instant as RFC3339 in the local timezone, or "0" when
// unset. Non-JSON encoders (TOON, fmt) render the value through this method.
func (t Timestamp) String() string {
	if t == 0 {
		return "0"
	}
	return t.Time().In(time.Local).Format(time.RFC3339)
}

// MarshalJSON renders a non-zero value as a quoted RFC3339 string in the local
// timezone; zero renders as the bare integer 0.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t == 0 {
		return []byte("0"), nil
	}
	return []byte(strconv.Quote(t.String())), nil
}

// UnmarshalJSON accepts a numeric Unix-seconds epoch, a quoted integer, an
// RFC3339 string, or null (→ 0).
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	n, err := parseEpochOrRFC3339(b, time.Second)
	if err != nil {
		return err
	}
	*t = Timestamp(n)
	return nil
}

// TimestampMilli is a Unix-milliseconds instant. It has the same rendering
// contract as Timestamp (RFC3339 out, epoch-or-RFC3339 in, zero→0); only the
// wire unit differs.
type TimestampMilli int64

// Time returns the instant as a time.Time.
func (t TimestampMilli) Time() time.Time { return time.UnixMilli(int64(t)) }

// Unix returns the raw wire value (milliseconds since the Unix epoch).
func (t TimestampMilli) Unix() int64 { return int64(t) }

// IsZero reports whether the value is the unset sentinel (0).
func (t TimestampMilli) IsZero() bool { return t == 0 }

// String renders the instant as RFC3339Nano in the local timezone (preserving
// sub-second precision), or "0" when unset. Non-JSON encoders (TOON, fmt)
// render the value through this method.
func (t TimestampMilli) String() string {
	if t == 0 {
		return "0"
	}
	return t.Time().In(time.Local).Format(time.RFC3339Nano)
}

// MarshalJSON renders a non-zero value as a quoted RFC3339 string in the local
// timezone; zero renders as the bare integer 0. RFC3339Nano is used so that
// sub-second (millisecond) precision survives a marshal→unmarshal round-trip;
// it elides trailing zeros, so whole-second values render identically to a
// plain RFC3339 timestamp.
func (t TimestampMilli) MarshalJSON() ([]byte, error) {
	if t == 0 {
		return []byte("0"), nil
	}
	return []byte(strconv.Quote(t.String())), nil
}

// UnmarshalJSON accepts a numeric Unix-milliseconds epoch, a quoted integer, an
// RFC3339 string, or null (→ 0).
func (t *TimestampMilli) UnmarshalJSON(b []byte) error {
	n, err := parseEpochOrRFC3339(b, time.Millisecond)
	if err != nil {
		return err
	}
	*t = TimestampMilli(n)
	return nil
}

// parseEpochOrRFC3339 decodes a JSON token into a wire integer of the given unit
// (time.Second or time.Millisecond). Accepts null/empty → 0, a bare or quoted
// integer (returned as-is), or a quoted RFC3339 string (converted to the unit).
func parseEpochOrRFC3339(b []byte, unit time.Duration) (int64, error) {
	s := string(bytes.TrimSpace(b))
	if s == "" || s == "null" {
		return 0, nil
	}
	if s[0] == '"' {
		inner := s[1 : len(s)-1]
		if inner == "" {
			return 0, nil
		}
		if n, err := strconv.ParseInt(inner, 10, 64); err == nil {
			return n, nil
		}
		tm, err := time.Parse(time.RFC3339, inner)
		if err != nil {
			return 0, err
		}
		if unit == time.Millisecond {
			return tm.UnixMilli(), nil
		}
		return tm.Unix(), nil
	}
	return strconv.ParseInt(s, 10, 64)
}
