package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Param returns the named path parameter from the matched route, or "" if
// absent. Handlers should use this instead of importing chi directly.
func Param(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// Wildcard returns the value matched by a trailing "/*" catch-all, or "" if
// the route had no wildcard.
func Wildcard(r *http.Request) string {
	return chi.URLParam(r, "*")
}

// RoutePattern returns the matched route pattern (e.g. "/items/{id}"),
// useful for metrics and structured logging. Returns "" when called outside
// a matched route.
func RoutePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		return rctx.RoutePattern()
	}
	return ""
}
