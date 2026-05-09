package server

// Router wraps go-chi/chi behind the same surface area we had with the
// stdlib-mux router. We picked chi because:
//
//   - chi's trie-based matcher resolves overlapping wildcard patterns by
//     specificity instead of refusing to register them. Stdlib ServeMux
//     panics on patterns where neither is strictly more specific (e.g.
//     "/{a}/{b}/lit/{c}" vs "/{a}/{b}/{c}/lit"); chi picks one deterministically.
//   - middleware wraps the dispatch, so 404s, 405s, and CORS preflights flow
//     through the chain (matching what we already had).
//
// Handler-side compatibility: chi keeps URL params in its own context, but
// our handlers call r.PathValue("name"). withPathValues copies chi's params
// into r.SetPathValue right before the handler runs, so handler code is
// unchanged.

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Router struct {
	chi chi.Router
}

func NewRouter() *Router {
	return &Router{chi: chi.NewRouter()}
}

// Use appends middleware to this router's scope. chi panics if called after
// any route is registered on this scope.
func (r *Router) Use(mw ...func(http.Handler) http.Handler) {
	r.chi.Use(mw...)
}

// Group creates a sub-router rooted at prefix. Middleware added inside fn
// applies only to routes registered on the sub-router.
func (r *Router) Group(prefix string, fn func(*Router)) {
	r.chi.Route(prefix, func(sub chi.Router) {
		fn(&Router{chi: sub})
	})
}

// With returns a sub-router with additional inline middleware.
func (r *Router) With(mw ...func(http.Handler) http.Handler) *Router {
	return &Router{chi: r.chi.With(mw...)}
}

// Handle registers handler for method+pattern. Pattern uses chi's syntax,
// which mirrors stdlib's {name} placeholders for single segments and adds a
// trailing /* for catch-all.
func (r *Router) Handle(method, pattern string, h http.Handler) {
	r.chi.Method(method, pattern, withPathValues(h))
}

func (r *Router) Get(p string, h http.HandlerFunc)    { r.Handle("GET", p, h) }
func (r *Router) Post(p string, h http.HandlerFunc)   { r.Handle("POST", p, h) }
func (r *Router) Put(p string, h http.HandlerFunc)    { r.Handle("PUT", p, h) }
func (r *Router) Delete(p string, h http.HandlerFunc) { r.Handle("DELETE", p, h) }
func (r *Router) Patch(p string, h http.HandlerFunc)  { r.Handle("PATCH", p, h) }

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.chi.ServeHTTP(w, req)
}

// withPathValues mirrors chi's URL params into r.SetPathValue so handlers
// can keep using stdlib's r.PathValue("name") regardless of the router.
func withPathValues(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			for i, k := range rctx.URLParams.Keys {
				if k == "*" {
					continue
				}
				r.SetPathValue(k, rctx.URLParams.Values[i])
			}
		}
		h.ServeHTTP(w, r)
	})
}
