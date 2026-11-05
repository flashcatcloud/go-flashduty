package flashduty

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestArtifactDownloadReturnsFileBytesVerbatim(t *testing.T) {
	// A JSON file shaped like an error envelope: the typed path would decode it
	// as an API error; Download must hand it back untouched.
	file := []byte(`{"error":{"code":"InvalidParameter","message":"not an API error"}}`)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/safari/artifact/stream" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("t") != "TOK" || q.Get("mode") != "download" || q.Get("app_key") != "KEY" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="data.json"`)
		_, _ = w.Write(file)
	})

	rc, resp, err := c.Artifacts.Download(context.Background(), "TOK")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer func() { _ = rc.Close() }()
	got, _ := io.ReadAll(rc)
	if !bytes.Equal(got, file) {
		t.Fatalf("body = %q, want %q", got, file)
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != `attachment; filename="data.json"` {
		t.Fatalf("Content-Disposition = %q", cd)
	}
}

func TestArtifactDownloadIsNotCappedBySizeOrClientTimeout(t *testing.T) {
	big := bytes.Repeat([]byte("x"), maxResponseBodySize+1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(big[:1024])
		w.(http.Flusher).Flush()
		time.Sleep(150 * time.Millisecond) // longer than the client Timeout below
		_, _ = w.Write(big[1024:])
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient("KEY", WithBaseURL(srv.URL), WithLogger(noopLogger{}), WithTimeout(50*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	rc, _, err := c.Artifacts.Download(context.Background(), "TOK")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer func() { _ = rc.Close() }()
	n, err := io.Copy(io.Discard, rc)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if n != int64(len(big)) {
		t.Fatalf("read %d bytes, want %d", n, len(big))
	}
}

func TestArtifactDownloadSurfacesErrorEnvelope(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		_, _ = io.WriteString(w, `{"request_id":"RID","error":{"code":"ResourceNotFound","message":"runner offline"}}`)
	})

	rc, _, err := c.Artifacts.Download(context.Background(), "TOK")
	if rc != nil {
		t.Fatal("want nil ReadCloser on error")
	}
	var apiErr *ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Message != "runner offline" || apiErr.RequestID != "RID" {
		t.Fatalf("err = %#v", err)
	}
}
