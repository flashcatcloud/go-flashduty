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
	if c.BaseURL.String() != "https://example.test" {
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
