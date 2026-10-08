package flashduty

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type seenRequest struct {
	auth   string
	hasKey bool
	key    string
}

func newCredentialClient(t *testing.T, accessToken bool) (*Client, *seenRequest) {
	t.Helper()
	seen := &seenRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		seen.auth = r.Header.Get("Authorization")
		seen.hasKey = r.URL.Query().Has("app_key")
		seen.key = r.URL.Query().Get("app_key")
		_, _ = io.WriteString(w, `{"request_id":"R","data":{}}`)
	}))
	t.Cleanup(srv.Close)
	var (
		c   *Client
		err error
	)
	if accessToken {
		c, err = NewClientWithAccessToken("tok", WithBaseURL(srv.URL), WithLogger(noopLogger{}))
	} else {
		c, err = NewClient("KEY", WithBaseURL(srv.URL), WithLogger(noopLogger{}))
	}
	if err != nil {
		t.Fatal(err)
	}
	return c, seen
}

func TestNewClientWithAccessTokenRequiresToken(t *testing.T) {
	if _, err := NewClientWithAccessToken(""); err == nil {
		t.Fatal("expected error for empty access token")
	}
}

func TestAccessTokenClientSendsBearerOnEveryRequestKind(t *testing.T) {
	ctx := context.Background()
	kinds := map[string]func(c *Client) error{
		"json": func(c *Client) error {
			_, err := c.do(ctx, "/incident/list", map[string]any{"p": 1}, nil)
			return err
		},
		"get": func(c *Client) error {
			_, err := c.doGet(ctx, "/incident/info", nil, nil)
			return err
		},
		"upload": func(c *Client) error {
			_, err := c.uploadFile(ctx, "/x/upload", url.Values{"a": {"b"}}, map[string]string{"f": "v"}, "a.txt", strings.NewReader("data"), nil)
			return err
		},
	}
	for name, call := range kinds {
		t.Run(name, func(t *testing.T) {
			c, seen := newCredentialClient(t, true)
			if err := call(c); err != nil {
				t.Fatal(err)
			}
			if seen.auth != "Bearer tok" {
				t.Errorf("Authorization = %q", seen.auth)
			}
			if seen.hasKey {
				t.Errorf("app_key must not be sent, got %q", seen.key)
			}
		})
	}
}

func TestNewClientStillSendsAppKey(t *testing.T) {
	ctx := context.Background()
	kinds := map[string]func(c *Client) error{
		"json": func(c *Client) error {
			_, err := c.do(ctx, "/incident/list", nil, nil)
			return err
		},
		"upload": func(c *Client) error {
			_, err := c.uploadFile(ctx, "/x/upload", nil, nil, "a.txt", strings.NewReader("data"), nil)
			return err
		},
	}
	for name, call := range kinds {
		t.Run(name, func(t *testing.T) {
			c, seen := newCredentialClient(t, false)
			if err := call(c); err != nil {
				t.Fatal(err)
			}
			if seen.key != "KEY" || seen.auth != "" {
				t.Errorf("app_key = %q, Authorization = %q", seen.key, seen.auth)
			}
		})
	}
}

// The per-request trigger token is the only credential on the without-app-key
// path, for both client kinds.
func TestWithoutAppKeyPathUsesOnlyPerRequestBearer(t *testing.T) {
	for _, accessToken := range []bool{false, true} {
		c, seen := newCredentialClient(t, accessToken)
		_, err := c.doMethodWithoutAppKey(context.Background(), http.MethodPost, "/safari/automation/triggers/t/fire", nil, nil, func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer trigger")
		})
		if err != nil {
			t.Fatal(err)
		}
		if seen.auth != "Bearer trigger" || seen.hasKey {
			t.Errorf("accessToken=%v: Authorization = %q, app_key present = %v", accessToken, seen.auth, seen.hasKey)
		}
	}
}
