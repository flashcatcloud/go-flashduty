package flashduty

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	// defaultMaxLogBodySize is the maximum size of a body before it is truncated in logs.
	defaultMaxLogBodySize = 2048
	// defaultLogPreviewSize is the size of the preview shown for truncated log content.
	defaultLogPreviewSize = 500
)

// sanitizeURL removes sensitive query parameters from a URL for safe logging.
func sanitizeURL(u *url.URL) string {
	sanitized := *u
	q := sanitized.Query()
	if q.Has("app_key") {
		q.Set("app_key", "[REDACTED]")
		sanitized.RawQuery = q.Encode()
	}
	return sanitized.String()
}

// sensitiveBodyKeys enumerates normalized JSON keys whose values must be
// redacted before bodies are logged. The set covers common credential aliases
// seen in API payloads and echoed error responses.
var sensitiveBodyKeys = map[string]struct{}{
	"apikey": {}, "xapikey": {}, "accesskey": {}, "password": {}, "passwd": {}, "pwd": {},
	"token": {}, "accesstoken": {}, "refreshtoken": {}, "idtoken": {}, "sessiontoken": {},
	"authtoken": {}, "oauthtoken": {}, "bearertoken": {}, "authorization": {}, "auth": {},
	"secret": {}, "clientsecret": {}, "secretkey": {}, "privatekey": {}, "signingkey": {},
	"credential": {}, "credentials": {},
}

// redactChildrenKeys enumerates normalized JSON keys whose nested values are
// always redacted regardless of inner key name. These containers (env, headers)
// hold user-chosen keys that frequently carry credentials, so the allow-list
// approach in sensitiveBodyKeys cannot catch them.
var redactChildrenKeys = map[string]struct{}{
	"env":     {},
	"headers": {},
}

// sanitizeBody redacts values of well-known sensitive JSON keys so that secrets
// do not appear in request/response logs. It is best-effort: empty or non-JSON
// bodies pass through unchanged.
func sanitizeBody(body string) string {
	if body == "" {
		return body
	}
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return body
	}
	sanitized, redacted := sanitizeJSONValue(v)
	if !redacted {
		return body
	}
	out, err := json.Marshal(sanitized)
	if err != nil {
		return body
	}
	return string(out)
}

func sanitizeJSONValue(v any) (any, bool) {
	switch value := v.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(value))
		redacted := false
		for key, item := range value {
			if isSensitiveBodyKey(key) {
				sanitized[key] = "[REDACTED]"
				redacted = true
				continue
			}
			if shouldRedactChildren(key) {
				sanitized[key] = redactAllLeaves(item)
				redacted = true
				continue
			}
			sanitizedItem, itemRedacted := sanitizeJSONValue(item)
			sanitized[key] = sanitizedItem
			redacted = redacted || itemRedacted
		}
		return sanitized, redacted
	case []any:
		sanitized := make([]any, len(value))
		redacted := false
		for i, item := range value {
			sanitizedItem, itemRedacted := sanitizeJSONValue(item)
			sanitized[i] = sanitizedItem
			redacted = redacted || itemRedacted
		}
		return sanitized, redacted
	default:
		return v, false
	}
}

// redactAllLeaves walks v and replaces every non-container leaf with
// "[REDACTED]", preserving the surrounding map/slice shape.
func redactAllLeaves(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			out[key] = redactAllLeaves(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = redactAllLeaves(item)
		}
		return out
	default:
		return "[REDACTED]"
	}
}

func isSensitiveBodyKey(key string) bool {
	_, ok := sensitiveBodyKeys[normalizeSensitiveBodyKey(key)]
	return ok
}

func shouldRedactChildren(key string) bool {
	_, ok := redactChildrenKeys[normalizeSensitiveBodyKey(key)]
	return ok
}

func normalizeSensitiveBodyKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range strings.ToLower(strings.TrimSpace(key)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sanitizeError removes a potential app_key-bearing URL from error messages.
func sanitizeError(err error) string {
	errStr := err.Error()
	idx := strings.Index(errStr, "app_key=")
	if idx == -1 {
		return errStr
	}
	endIdx := strings.IndexAny(errStr[idx:], "& ")
	if endIdx == -1 {
		return errStr[:idx] + "app_key=[REDACTED]"
	}
	return errStr[:idx] + "app_key=[REDACTED]" + errStr[idx+endIdx:]
}

// truncateBody truncates a body string if it exceeds the default max log size.
func truncateBody(body string) string {
	bodyLen := len(body)
	if bodyLen <= defaultMaxLogBodySize {
		return body
	}
	previewSize := defaultLogPreviewSize
	if previewSize > bodyLen {
		previewSize = bodyLen
	}
	return fmt.Sprintf("[LARGE_BODY: truncated, size: %d bytes, preview: %s...]", bodyLen, body[:previewSize])
}
