package flashduty

import (
	"net/http"
	"strconv"
	"time"
)

// RateLimit captures rate-limit signals parsed from a response, best-effort.
// Each field is zero when the server did not send the corresponding header.
type RateLimit struct {
	Limit      int           // X-RateLimit-Limit, if present
	Remaining  int           // X-RateLimit-Remaining, if present
	Reset      time.Time     // X-RateLimit-Reset (unix seconds), if present
	RetryAfter time.Duration // Retry-After (delta-seconds), if present
}

func parseRateLimit(h http.Header) RateLimit {
	var rl RateLimit
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			rl.RetryAfter = time.Duration(secs) * time.Second
		}
	}
	if v := h.Get("X-RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.Limit = n
		}
	}
	if v := h.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.Remaining = n
		}
	}
	if v := h.Get("X-RateLimit-Reset"); v != "" {
		if secs, err := strconv.ParseInt(v, 10, 64); err == nil {
			rl.Reset = time.Unix(secs, 0)
		}
	}
	return rl
}
