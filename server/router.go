package server

// https://github.com/go-chi/chi/blob/master/middleware/realip.go

import "net/http"

type Router struct {
	mux        *http.ServeMux
	middleware []func(http.Handler) http.Handler
	prefix     string
}

func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (r *Router) Use(mw ...func(http.Handler) http.Handler) {
	r.middleware = append(r.middleware, mw...)
}

func (r *Router) Group(prefix string, fn func(*Router)) {
	sub := &Router{
		mux:        r.mux, // shared mux
		middleware: append([]func(http.Handler) http.Handler{}, r.middleware...),
		prefix:     r.prefix + prefix,
	}
	fn(sub)
}

func (r *Router) With(mw ...func(http.Handler) http.Handler) *Router {
	return &Router{
		mux:        r.mux,
		middleware: append(append([]func(http.Handler) http.Handler{}, r.middleware...), mw...),
		prefix:     r.prefix,
	}
}

func (r *Router) handle(method, path string, h http.Handler) {
	handler := h
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}
	r.mux.Handle(method+" "+r.prefix+path, handler)
}

// Handle registers handler for method+pattern, satisfying the Router interface
// from routes.go. Pattern uses net/http 1.22+ {name} placeholder syntax.
func (r *Router) Handle(method, pattern string, h http.Handler) {
	r.handle(method, pattern, h)
}

func (r *Router) Get(p string, h http.HandlerFunc)    { r.handle("GET", p, h) }
func (r *Router) Post(p string, h http.HandlerFunc)   { r.handle("POST", p, h) }
func (r *Router) Put(p string, h http.HandlerFunc)    { r.handle("PUT", p, h) }
func (r *Router) Delete(p string, h http.HandlerFunc) { r.handle("DELETE", p, h) }
func (r *Router) Patch(p string, h http.HandlerFunc)  { r.handle("PATCH", p, h) }

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
