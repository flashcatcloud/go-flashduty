//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	flashduty "github.com/flashcatcloud/go-flashduty"
)

// newClient builds a client from the E2E environment, skipping the test when no
// app key is configured. It honors FLASHDUTY_E2E_APP_KEY / FLASHDUTY_E2E_BASE_URL
// and falls back to FLASHDUTY_APP_KEY / FLASHDUTY_BASE_URL.
func newClient(t *testing.T) *flashduty.Client {
	t.Helper()
	key := firstEnv("FLASHDUTY_E2E_APP_KEY", "FLASHDUTY_APP_KEY")
	if key == "" {
		t.Skip("set FLASHDUTY_E2E_APP_KEY (and optionally FLASHDUTY_E2E_BASE_URL) to run E2E tests")
	}
	opts := []flashduty.Option{flashduty.WithLogger(noopLogger{})}
	if base := firstEnv("FLASHDUTY_E2E_BASE_URL", "FLASHDUTY_BASE_URL"); base != "" {
		opts = append(opts, flashduty.WithBaseURL(base))
	}
	c, err := flashduty.NewClient(key, opts...)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return c
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

var nameCounter int64

// uniqueName returns a collision-resistant name for a test-created resource. All
// E2E-created resources carry the "gofd-e2e-" prefix so they are identifiable
// and can be swept if a cleanup is ever missed.
func uniqueName(prefix string) string {
	n := atomic.AddInt64(&nameCounter, 1)
	return fmt.Sprintf("gofd-e2e-%s-%d-%d", prefix, time.Now().UnixMicro(), n)
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}
