package http

// Logger is the logging surface the client writes to. The client emits one
// Debug record per completed request and nothing else, so a single method is
// the whole surface. It is satisfied structurally by github.com/toaweme/log's
// Slog and by *slog.Logger, so callers inject either without an adapter.
// There is deliberately no With/handler method, and per-request context is
// passed as args on the call instead.
type Logger interface {
	Debug(msg string, args ...any)
}

// nopLogger is the default Logger: it discards everything, so the client is
// silent unless a logger is injected via WithLogger.
type nopLogger struct{}

var _ Logger = nopLogger{}

func (nopLogger) Debug(string, ...any) {}
