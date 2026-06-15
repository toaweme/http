package http

// Logger is the minimal leveled logging surface the client writes to. It is
// satisfied structurally by github.com/toaweme/log's Slog, so callers can
// inject that directly, or a null logger to discard output. There is
// deliberately no With/handler method so external loggers satisfy it without
// an adapter; per-request context is passed as args on each call instead.
type Logger interface {
	Trace(msg string, args ...any)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// nopLogger is the default Logger: it discards everything, so the client is
// silent unless a logger is injected via WithLogger.
type nopLogger struct{}

var _ Logger = nopLogger{}

func (nopLogger) Trace(string, ...any) {}
func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}
