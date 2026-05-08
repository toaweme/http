package server

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// tag wraps a handler so it appends a marker to a shared trace, letting tests
// assert middleware ordering.
func tag(name string, trace *[]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*trace = append(*trace, name)
			next.ServeHTTP(w, r)
		})
	}
}

func equalSlice(t *testing.T, got, want []string, msg string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %v want %v", msg, got, want)
	}
}

func TestRouter_RootMiddlewareWrapsMux(t *testing.T) {
	t.Run("runs on matched routes", func(t *testing.T) {
		var trace []string
		r := NewRouter()
		r.Use(tag("root", &trace))
		r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
			trace = append(trace, "handler")
			w.WriteHeader(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", "/foo", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200", rec.Code)
		}
		equalSlice(t, trace, []string{"root", "handler"}, "trace")
	})

	t.Run("runs on unmatched method (e.g. CORS preflight)", func(t *testing.T) {
		// regression: ServeMux returns 405 for OPTIONS on a GET-only route. Root
		// middleware must still see the request — that's how CORS preflights work.
		var trace []string
		r := NewRouter()
		r.Use(tag("root", &trace))
		r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {})

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("OPTIONS", "/foo", nil))

		equalSlice(t, trace, []string{"root"}, "root middleware should run on 405s")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status: got %d want 405", rec.Code)
		}
	})

	t.Run("runs on 404", func(t *testing.T) {
		var trace []string
		r := NewRouter()
		r.Use(tag("root", &trace))
		r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {})

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", "/nope", nil))

		equalSlice(t, trace, []string{"root"}, "trace")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status: got %d want 404", rec.Code)
		}
	})
}

func TestRouter_GroupScopedMiddleware(t *testing.T) {
	var trace []string
	r := NewRouter()
	r.Use(tag("root", &trace))

	r.Group("/api", func(g *Router) {
		g.Use(tag("scoped", &trace))
		g.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
			trace = append(trace, "handler")
		})
	})

	r.Get("/bare", func(w http.ResponseWriter, _ *http.Request) {
		trace = append(trace, "bare")
	})

	t.Run("scoped middleware runs on group routes", func(t *testing.T) {
		trace = nil
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", "/api/foo", nil))
		equalSlice(t, trace, []string{"root", "scoped", "handler"}, "trace")
	})

	t.Run("scoped middleware does NOT run on routes outside the group", func(t *testing.T) {
		trace = nil
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", "/bare", nil))
		equalSlice(t, trace, []string{"root", "bare"}, "trace")
	})
}

func TestRouter_WithScopedMiddleware(t *testing.T) {
	var trace []string
	r := NewRouter()
	r.Use(tag("root", &trace))

	r.With(tag("with", &trace)).Get("/scoped", func(w http.ResponseWriter, _ *http.Request) {
		trace = append(trace, "scoped-handler")
	})
	r.Get("/plain", func(w http.ResponseWriter, _ *http.Request) {
		trace = append(trace, "plain-handler")
	})

	trace = nil
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/scoped", nil))
	equalSlice(t, trace, []string{"root", "with", "scoped-handler"}, "scoped trace")

	trace = nil
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/plain", nil))
	equalSlice(t, trace, []string{"root", "plain-handler"}, "plain trace")
}

func TestRouter_NestedGroups(t *testing.T) {
	var trace []string
	r := NewRouter()
	r.Use(tag("root", &trace))

	r.Group("/a", func(g *Router) {
		g.Use(tag("a", &trace))
		g.Group("/b", func(gg *Router) {
			gg.Use(tag("b", &trace))
			gg.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
				trace = append(trace, "handler")
			})
		})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/a/b/foo", nil))
	equalSlice(t, trace, []string{"root", "a", "b", "handler"}, "nested trace")
}

func mustPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		got := recover()
		if got == nil {
			t.Fatalf("expected panic %q, got none", want)
		}
		if s, _ := got.(string); s != want {
			t.Fatalf("panic: got %q want %q", got, want)
		}
	}()
	fn()
}

func TestRouter_UseAfterRoutePanics(t *testing.T) {
	const msg = "server: all middleware must be registered before routes on this router scope"

	t.Run("root", func(t *testing.T) {
		r := NewRouter()
		r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {})
		mustPanic(t, msg, func() { r.Use(func(h http.Handler) http.Handler { return h }) })
	})

	t.Run("subrouter", func(t *testing.T) {
		r := NewRouter()
		r.Group("/api", func(g *Router) {
			g.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {})
			mustPanic(t, msg, func() { g.Use(func(h http.Handler) http.Handler { return h }) })
		})
	})

	t.Run("root after subrouter route", func(t *testing.T) {
		r := NewRouter()
		r.Group("/api", func(g *Router) {
			g.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {})
		})
		mustPanic(t, msg, func() { r.Use(func(h http.Handler) http.Handler { return h }) })
	})
}

func TestRouter_NoMiddleware(t *testing.T) {
	r := NewRouter()
	r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/foo", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
		t.Fatalf("body: got %q want %q", got, "ok")
	}
}
