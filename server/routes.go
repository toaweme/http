package server

import "net/http"

// Route is a single HTTP route definition.
// Pattern uses net/http 1.22+ placeholder syntax: {name}.
type Route struct {
	Method  string
	Pattern string
	Handler http.Handler
}

// HandleRouter is the minimal interface satisfied by any HTTP framework that
// can register a method+pattern+handler triple. *Router satisfies it.
type HandleRouter interface {
	Handle(method, pattern string, handler http.Handler)
}

// Register registers all routes on r.
func Register(r HandleRouter, routes []Route) {
	for _, rt := range routes {
		r.Handle(rt.Method, rt.Pattern, rt.Handler)
	}
}
