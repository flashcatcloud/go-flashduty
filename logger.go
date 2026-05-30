package flashduty

import "log/slog"

// Logger defines the logging interface for the SDK.
// Consumers can implement this to integrate with any logging backend.
//
// The keysAndValues parameter uses alternating key-value pairs (slog-style):
//
//	logger.Info("request complete", "status", 200, "duration_ms", 42)
//
// To adapt logrus, implement a thin wrapper that converts keysAndValues to logrus.Fields:
//
//	type logrusAdapter struct{ *logrus.Logger }
//	func (a *logrusAdapter) Info(msg string, kv ...any)  { a.WithFields(kvToFields(kv)).Info(msg) }
//	func (a *logrusAdapter) Warn(msg string, kv ...any)  { a.WithFields(kvToFields(kv)).Warn(msg) }
//	func (a *logrusAdapter) Error(msg string, kv ...any) { a.WithFields(kvToFields(kv)).Error(msg) }
//	func (a *logrusAdapter) Debug(msg string, kv ...any) { a.WithFields(kvToFields(kv)).Debug(msg) }
//	func kvToFields(kv []any) logrus.Fields {
//	    fields := make(logrus.Fields, len(kv)/2)
//	    for i := 0; i+1 < len(kv); i += 2 {
//	        if key, ok := kv[i].(string); ok {
//	            fields[key] = kv[i+1]
//	        }
//	    }
//	    return fields
//	}
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

// defaultLogger is the package-level default backed by log/slog.
var defaultLogger Logger = &slogLogger{}

// slogLogger wraps log/slog as the default Logger implementation.
type slogLogger struct{}

func (s *slogLogger) Debug(msg string, keysAndValues ...any) { slog.Debug(msg, keysAndValues...) }
func (s *slogLogger) Info(msg string, keysAndValues ...any)  { slog.Info(msg, keysAndValues...) }
func (s *slogLogger) Warn(msg string, keysAndValues ...any)  { slog.Warn(msg, keysAndValues...) }
func (s *slogLogger) Error(msg string, keysAndValues ...any) { slog.Error(msg, keysAndValues...) }
