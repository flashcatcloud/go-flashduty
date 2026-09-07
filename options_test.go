package flashduty

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClientDefaultsAndOptions(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected error for empty app key")
	}
	c, err := NewClient("KEY",
		WithBaseURL("https://example.test"),
		WithTimeout(5*time.Second),
		WithUserAgent("ua/1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL.String() != "https://example.test/" {
		t.Fatalf("BaseURL = %s", c.BaseURL)
	}
	if c.UserAgent != "ua/1" {
		t.Fatalf("UserAgent = %s", c.UserAgent)
	}
	if c.client.Timeout != 5*time.Second {
		t.Fatalf("timeout = %s", c.client.Timeout)
	}
}

func TestWithBaseURLInvalidReturnsError(t *testing.T) {
	if _, err := NewClient("KEY", WithBaseURL("://bad")); err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}

func TestWithBaseURLPreservesPathPrefix(t *testing.T) {
	for _, tc := range []struct {
		base string
		want string
	}{
		{"https://example.test", "https://example.test/rum/data/query"},
		{"https://example.test/", "https://example.test/rum/data/query"},
		{"https://example.test/api", "https://example.test/api/rum/data/query"},
		{"https://example.test/api/", "https://example.test/api/rum/data/query"},
		{"https://example.test/a/b", "https://example.test/a/b/rum/data/query"},
		{"http://192.0.2.10:12345", "http://192.0.2.10:12345/rum/data/query"},
		{"http://192.0.2.10:12345/api", "http://192.0.2.10:12345/api/rum/data/query"},
	} {
		c, err := NewClient("KEY", WithBaseURL(tc.base))
		if err != nil {
			t.Fatalf("NewClient(%q): %v", tc.base, err)
		}
		req, err := c.newRequest(t.Context(), http.MethodPost, "/rum/data/query", nil)
		if err != nil {
			t.Fatalf("newRequest with base %q: %v", tc.base, err)
		}
		req.URL.RawQuery = ""
		if got := req.URL.String(); got != tc.want {
			t.Errorf("base %q: request URL = %q, want %q", tc.base, got, tc.want)
		}
	}
}

func TestWithHTTPClientNilIgnored(t *testing.T) {
	c, err := NewClient("KEY", WithHTTPClient(nil))
	if err != nil || c.client == nil {
		t.Fatalf("nil http client must be ignored, got err=%v client=%v", err, c.client)
	}
}

type markerRT struct{ http.RoundTripper }

func TestWithTransportSetsRoundTripper(t *testing.T) {
	rt := &markerRT{}
	c, err := NewClient("KEY", WithTransport(rt))
	if err != nil || c.client.Transport != rt {
		t.Fatalf("WithTransport not applied: err=%v transport=%v", err, c.client.Transport)
	}
	c2, _ := NewClient("KEY", WithTransport(nil))
	if c2.client.Transport != nil {
		t.Fatalf("nil transport should be ignored")
	}
}
