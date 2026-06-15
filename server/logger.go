package server

// Logger is the minimal leveled logging surface the server writes to. It is
// satisfied structurally by github.com/toaweme/log's Slog, so callers can
// inject that directly, or a null logger to discard output. It is defined here
// rather than imported from the client module so the server module never
// depends on its parent; a single concrete logger satisfies both.
type Logger interface {
	Trace(msg string, args ...any)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
