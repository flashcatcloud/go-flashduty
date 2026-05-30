package flashduty

import "testing"

type captureLogger struct{ msgs []string }

func (c *captureLogger) Debug(msg string, kv ...any) { c.msgs = append(c.msgs, msg) }
func (c *captureLogger) Info(msg string, kv ...any)  { c.msgs = append(c.msgs, msg) }
func (c *captureLogger) Warn(msg string, kv ...any)  { c.msgs = append(c.msgs, msg) }
func (c *captureLogger) Error(msg string, kv ...any) { c.msgs = append(c.msgs, msg) }

func TestDefaultLoggerImplementsInterface(t *testing.T) {
	var _ Logger = defaultLogger
	var _ Logger = (*captureLogger)(nil)
}
