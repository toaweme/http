package server

import (
	"context"
	"io"
	"net/http"
)

// bodyLimitKey carries the request body as it arrived, before any limit wrapped
// it. A route-level limit needs the original, because a wider limit layered on
// top of a narrower one still reads through the narrower reader and the body is
// cut at the smaller size.
type bodyLimitKey struct{}

// LimitBody returns a middleware capping how many bytes a handler may read from
// a request body.
//
// The cap is enforced as the body is read rather than from Content-Length, since
// a chunked request declares no length at all and a declared one is only the
// client's word. A handler reading past the cap gets a *http.MaxBytesError, which
// is the error a read site maps to whatever refusal its surface answers with.
//
// A limit of zero or less leaves the body alone.
func LimitBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, LimitRequestBody(w, r, limit))
		})
	}
}

// LimitRequestBody caps r's body at limit and returns the request to serve on.
//
// It is the per-route override of a router-wide LimitBody. The cap replaces what
// a wider scope applied rather than nesting under it, so a route that accepts
// more bytes than the router's default actually gets them.
func LimitRequestBody(w http.ResponseWriter, r *http.Request, limit int64) *http.Request {
	if r == nil || r.Body == nil || limit <= 0 {
		return r
	}
	body := r.Body
	if original, ok := r.Context().Value(bodyLimitKey{}).(io.ReadCloser); ok {
		body = original
	} else {
		r = r.WithContext(context.WithValue(r.Context(), bodyLimitKey{}, body))
	}
	r.Body = http.MaxBytesReader(w, body, limit)
	return r
}
