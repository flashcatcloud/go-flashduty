package flashduty

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// withLocal temporarily pins time.Local so timezone-dependent rendering is
// deterministic, then restores it.
func withLocal(t *testing.T, name string) {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location %s: %v", name, err)
	}
	orig := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = orig })
}

func TestTimestamp_MarshalJSON_RendersRFC3339InLocalTZ(t *testing.T) {
	withLocal(t, "Asia/Shanghai") // UTC+8
	ts := Timestamp(time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).Unix())
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// 08:00Z rendered in +08:00 is 16:00+08:00
	if want := `"2026-05-28T16:00:00+08:00"`; string(b) != want {
		t.Errorf("MarshalJSON = %s, want %s", b, want)
	}
}

func TestTimestamp_MarshalJSON_ZeroStaysNumericZero(t *testing.T) {
	b, err := json.Marshal(Timestamp(0))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "0" {
		t.Errorf("zero Timestamp = %s, want 0 (unset sentinel, never a 1970 date)", b)
	}
}

func TestTimestamp_UnmarshalJSON_FromEpochNumber(t *testing.T) {
	var ts Timestamp
	if err := json.Unmarshal([]byte("1748419200"), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ts.Unix() != 1748419200 {
		t.Errorf("Unix() = %d, want 1748419200", ts.Unix())
	}
}

func TestTimestamp_UnmarshalJSON_FromRFC3339String(t *testing.T) {
	want := time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).Unix()
	var ts Timestamp
	if err := json.Unmarshal([]byte(`"2026-05-28T08:00:00Z"`), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ts.Unix() != want {
		t.Errorf("Unix() = %d, want %d", ts.Unix(), want)
	}
}

func TestTimestamp_UnmarshalJSON_FromNull(t *testing.T) {
	ts := Timestamp(123)
	if err := json.Unmarshal([]byte("null"), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ts != 0 {
		t.Errorf("null -> %d, want 0", ts)
	}
}

func TestTimestamp_RoundTrip(t *testing.T) {
	withLocal(t, "Asia/Shanghai")
	for _, epoch := range []int64{0, 1, 1748419200} {
		ts := Timestamp(epoch)
		b, err := json.Marshal(ts)
		if err != nil {
			t.Fatalf("marshal %d: %v", epoch, err)
		}
		var back Timestamp
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if back != ts {
			t.Errorf("round-trip %d: got %d (via %s)", epoch, back, b)
		}
	}
}

func TestTimestamp_OmitemptyDropsZero(t *testing.T) {
	type wrap struct {
		AckTime Timestamp `json:"ack_time,omitempty"`
	}
	b, err := json.Marshal(wrap{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "{}" {
		t.Errorf("omitempty zero Timestamp not dropped: %s", b)
	}
}

func TestTimestamp_Helpers(t *testing.T) {
	ts := Timestamp(1748419200)
	if ts.Unix() != 1748419200 {
		t.Errorf("Unix() = %d, want 1748419200", ts.Unix())
	}
	if !ts.Time().Equal(time.Unix(1748419200, 0)) {
		t.Errorf("Time() = %v, want %v", ts.Time(), time.Unix(1748419200, 0))
	}
	if !Timestamp(0).IsZero() {
		t.Errorf("Timestamp(0).IsZero() = false, want true")
	}
	if ts.IsZero() {
		t.Errorf("non-zero IsZero() = true, want false")
	}
}

func TestTimestamp_InStructRendersRFC3339(t *testing.T) {
	withLocal(t, "UTC")
	type incident struct {
		StartTime Timestamp `json:"start_time"`
	}
	b, err := json.Marshal(incident{StartTime: Timestamp(time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).Unix())})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `{"start_time":"2026-05-28T08:00:00Z"}`; string(b) != want {
		t.Errorf("struct marshal = %s, want %s", b, want)
	}
}

func TestTimestampMilli_MarshalJSON_RendersRFC3339InLocalTZ(t *testing.T) {
	withLocal(t, "Asia/Shanghai") // UTC+8
	ts := TimestampMilli(time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).UnixMilli())
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `"2026-05-28T16:00:00+08:00"`; string(b) != want {
		t.Errorf("MarshalJSON = %s, want %s", b, want)
	}
}

func TestTimestampMilli_MarshalJSON_ZeroStaysNumericZero(t *testing.T) {
	b, err := json.Marshal(TimestampMilli(0))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "0" {
		t.Errorf("zero TimestampMilli = %s, want 0", b)
	}
}

func TestTimestampMilli_UnmarshalJSON_FromEpochMillis(t *testing.T) {
	var ts TimestampMilli
	if err := json.Unmarshal([]byte("1779004800000"), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int64(ts) != 1779004800000 {
		t.Errorf("wire ms not preserved: got %d, want 1779004800000", int64(ts))
	}
}

func TestTimestampMilli_UnmarshalJSON_FromRFC3339String(t *testing.T) {
	want := time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).UnixMilli()
	var ts TimestampMilli
	if err := json.Unmarshal([]byte(`"2026-05-28T08:00:00Z"`), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int64(ts) != want {
		t.Errorf("ms = %d, want %d", int64(ts), want)
	}
}

func TestTimestampMilli_RoundTrip(t *testing.T) {
	withLocal(t, "Asia/Shanghai")
	for _, ms := range []int64{0, 1748419200000} {
		ts := TimestampMilli(ms)
		b, err := json.Marshal(ts)
		if err != nil {
			t.Fatalf("marshal %d: %v", ms, err)
		}
		var back TimestampMilli
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if back != ts {
			t.Errorf("round-trip %d: got %d (via %s)", ms, back, b)
		}
	}
}

func TestTimestampMilli_HelpersAndOmitempty(t *testing.T) {
	ts := TimestampMilli(1748419200000)
	if ts.Unix() != 1748419200000 {
		t.Errorf("Unix() = %d, want 1748419200000 (raw ms value)", ts.Unix())
	}
	if !ts.Time().Equal(time.UnixMilli(1748419200000)) {
		t.Errorf("Time() = %v, want %v", ts.Time(), time.UnixMilli(1748419200000))
	}
	if !TimestampMilli(0).IsZero() || ts.IsZero() {
		t.Errorf("IsZero wrong")
	}
	type wrap struct {
		CreatedAt TimestampMilli `json:"created_at,omitempty"`
	}
	b, _ := json.Marshal(wrap{})
	if string(b) != "{}" {
		t.Errorf("omitempty zero TimestampMilli not dropped: %s", b)
	}
}

func TestTimestampMilli_RoundTrip_PreservesFractionalMillis(t *testing.T) {
	withLocal(t, "UTC")
	ts := TimestampMilli(1748419200123) // .123 of a second
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), ".123") {
		t.Errorf("sub-second precision lost in marshal: %s", b)
	}
	var back TimestampMilli
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
	if back != ts {
		t.Errorf("fractional-ms round-trip lost data: got %d, want %d (via %s)", back, ts, b)
	}
}

func TestTimestamp_UnmarshalJSON_FromQuotedInteger(t *testing.T) {
	var ts Timestamp
	if err := json.Unmarshal([]byte(`"1748419200"`), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ts.Unix() != 1748419200 {
		t.Errorf("quoted integer: got %d, want 1748419200", ts.Unix())
	}
}

func TestTimestamp_UnmarshalJSON_FromEmptyString(t *testing.T) {
	ts := Timestamp(99)
	if err := json.Unmarshal([]byte(`""`), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ts != 0 {
		t.Errorf(`empty string -> %d, want 0`, ts)
	}
}
