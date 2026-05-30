package flashduty

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func TestAddQueryParams(t *testing.T) {
	type opts struct {
		PageID     int64    `url:"page_id"`
		Type       string   `url:"type"`
		StartAt    int64    `url:"start_at,omitempty"`
		Note       string   `url:"note,omitempty"`
		Enabled    bool     `url:"enabled,omitempty"`
		Tags       []string `url:"tags,omitempty"`
		Unexported int64    `url:"-"`
		Skipped    string
	}

	got, err := addQueryParams("/x", &opts{
		PageID:  7,
		Type:    "incident",
		Note:    "", // omitempty zero -> dropped
		Enabled: true,
		Tags:    []string{"a", "b"},
		Skipped: "ignored", // no url tag -> dropped
	})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(got)
	q := u.Query()

	if q.Get("page_id") != "7" {
		t.Errorf("page_id = %q, want 7", q.Get("page_id"))
	}
	if q.Get("type") != "incident" {
		t.Errorf("type = %q", q.Get("type"))
	}
	if _, ok := q["start_at"]; ok {
		t.Errorf("start_at should be omitted when zero, got %q", q.Get("start_at"))
	}
	if _, ok := q["note"]; ok {
		t.Errorf("note should be omitted when empty")
	}
	if q.Get("enabled") != "true" {
		t.Errorf("enabled = %q, want true", q.Get("enabled"))
	}
	if vals := q["tags"]; len(vals) != 2 || vals[0] != "a" || vals[1] != "b" {
		t.Errorf("tags = %v, want [a b]", vals)
	}
	if _, ok := q["-"]; ok {
		t.Errorf("url:\"-\" field must be skipped")
	}
	if _, ok := q["Skipped"]; ok {
		t.Errorf("untagged field must be skipped")
	}
}

func TestAddQueryParamsNilAndEmpty(t *testing.T) {
	if got, _ := addQueryParams("/x", nil); got != "/x" {
		t.Errorf("nil opt = %q, want /x", got)
	}
	type empty struct {
		A string `url:"a,omitempty"`
	}
	if got, _ := addQueryParams("/x", &empty{}); got != "/x" {
		t.Errorf("all-zero opt = %q, want /x", got)
	}
	var nilPtr *empty
	if got, _ := addQueryParams("/x", nilPtr); got != "/x" {
		t.Errorf("nil pointer opt = %q, want /x", got)
	}
}

func TestAddQueryParamsRejectsNonStruct(t *testing.T) {
	if _, err := addQueryParams("/x", "not a struct"); err == nil {
		t.Fatal("expected error for non-struct opt")
	}
}

func TestAddQueryParamsAppendsToExistingQuery(t *testing.T) {
	type opts struct {
		B string `url:"b"`
	}
	got, err := addQueryParams("/x?a=1", &opts{B: "2"})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("a") != "1" || q.Get("b") != "2" {
		t.Errorf("merged query = %q", got)
	}
}

func TestDoGetSendsQueryAndDecodes(t *testing.T) {
	var gotMethod, gotPath, gotPageID, gotAppKey string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotPageID = r.URL.Query().Get("page_id")
		gotAppKey = r.URL.Query().Get("app_key")
		_, _ = io.WriteString(w, `{"request_id":"RID","data":{"change_id":"c1"}}`)
	})

	type req struct {
		PageID int64 `url:"page_id"`
	}
	var out struct {
		ChangeID string `json:"change_id"`
	}
	resp, err := c.doGet(context.Background(), "/status-page/change/info", &req{PageID: 42}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/status-page/change/info" {
		t.Errorf("path = %s", gotPath)
	}
	if gotPageID != "42" {
		t.Errorf("page_id = %q, want 42", gotPageID)
	}
	if gotAppKey != "KEY" {
		t.Errorf("app_key = %q, want KEY", gotAppKey)
	}
	if out.ChangeID != "c1" || resp.RequestID != "RID" {
		t.Errorf("decode failed: out=%+v resp=%+v", out, resp)
	}
}
