package server

// Inspired by go-chi/chi.
//
// Two layers of middleware:
//
//   - Root middleware (added via Use on the *Router returned by NewRouter)
//     wraps the mux dispatch, so it sees every request — including 404s, 405s,
//     and CORS preflights to paths that only have method-specific handlers.
//
//   - Inline middleware (added via With/Group on a sub-router, or Use on a
//     sub-router inside a Group closure) wraps individual route handlers
//     before they are registered on the mux. It only runs for matched routes.
//
// Use panics if called after the router scope it targets has registered any
// route, matching chi's contract — middleware is frozen at first Handle.

import "net/http"

type Router struct {
	mux *http.ServeMux

	// middleware is the root chain. Populated only on the root router via
	// Use; ignored on sub-routers (sub Use writes to inline instead).
	middleware []func(http.Handler) http.Handler

	// inline is the scoped chain applied per-route in handle. Only meaningful
	// on sub-routers created via Group/With.
	inline []func(http.Handler) http.Handler

	prefix string
	isRoot bool

	// frozen flips true when this router scope registers its first route, and
	// is mirrored onto the root so that root.Use panics once any sub-router
	// has registered routes too.
	frozen bool

	// root points to the topmost router. On the root itself, root == r. Used
	// to share the compiled chain and the freeze signal across sub-routers.
	root *Router

	// handler is the compiled root chain wrapping mux. Only set on the root.
	handler http.Handler
}

func NewRouter() *Router {
	r := &Router{mux: http.NewServeMux(), isRoot: true}
	r.root = r
	return r
}

// Use appends middleware to this router's scope. On the root, middleware
// wraps the entire mux dispatch (so it sees unmatched requests too). On a
// sub-router, middleware wraps individual handlers registered on that sub.
// Panics if called after the scope has registered any route.
func (r *Router) Use(mw ...func(http.Handler) http.Handler) {
	if r.frozen {
		panic("server: all middleware must be registered before routes on this router scope")
	}
	if r.isRoot {
		r.middleware = append(r.middleware, mw...)
	} else {
		r.inline = append(r.inline, mw...)
	}
}

// Group creates a sub-router with the given path prefix. Middleware added via
// Use/With inside fn applies only to routes registered on the sub-router.
// The root middleware still wraps every dispatch.
func (r *Router) Group(prefix string, fn func(*Router)) {
	sub := &Router{
		mux:    r.mux,
		inline: append([]func(http.Handler) http.Handler{}, r.inline...),
		prefix: r.prefix + prefix,
		root:   r.root,
	}
	fn(sub)
}

// With returns a sub-router with additional inline middleware. Use to scope
// middleware to a single route or chained Get/Post call.
func (r *Router) With(mw ...func(http.Handler) http.Handler) *Router {
	return &Router{
		mux:    r.mux,
		inline: append(append([]func(http.Handler) http.Handler{}, r.inline...), mw...),
		prefix: r.prefix,
		root:   r.root,
	}
}

func (r *Router) handle(method, path string, h http.Handler) {
	r.frozen = true
	r.root.frozen = true
	if r.root.handler == nil {
		r.root.compile()
	}

	handler := h
	for i := len(r.inline) - 1; i >= 0; i-- {
		handler = r.inline[i](handler)
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

// compile freezes the root middleware chain. Called from handle on first
// route registration, or from ServeHTTP if no routes were ever registered
// (e.g. tests). Idempotent.
func (r *Router) compile() {
	if r.handler != nil {
		return
	}
	var h http.Handler = r.mux
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}
	r.handler = h
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.handler == nil {
		r.compile()
	}
	r.handler.ServeHTTP(w, req)
}
