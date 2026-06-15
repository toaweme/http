package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_Param(t *testing.T) {
	r := NewRouter()
	var got string
	r.Get("/items/{id}", func(w http.ResponseWriter, req *http.Request) {
		got = Param(req, "id")
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/42", http.NoBody))
	if got != "42" {
		t.Fatalf("Param: got %q want 42", got)
	}
}

func Test_Param_MissingKeyReturnsEmpty(t *testing.T) {
	r := NewRouter()
	var got = "sentinel"
	r.Get("/items/{id}", func(w http.ResponseWriter, req *http.Request) {
		got = Param(req, "nope")
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/42", http.NoBody))
	if got != "" {
		t.Fatalf("Param for absent key: got %q want empty", got)
	}
}

func Test_Wildcard(t *testing.T) {
	r := NewRouter()
	var got string
	r.Get("/files/*", func(w http.ResponseWriter, req *http.Request) {
		got = Wildcard(req)
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/a/b/c.txt", http.NoBody))
	if got != "a/b/c.txt" {
		t.Fatalf("Wildcard: got %q want a/b/c.txt", got)
	}
}

func Test_RoutePattern(t *testing.T) {
	r := NewRouter()
	var got string
	r.Get("/items/{id}", func(w http.ResponseWriter, req *http.Request) {
		got = RoutePattern(req)
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items/42", http.NoBody))
	if got != "/items/{id}" {
		t.Fatalf("RoutePattern: got %q want /items/{id}", got)
	}
}

func Test_RoutePattern_OutsideMatchedRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	if got := RoutePattern(req); got != "" {
		t.Fatalf("RoutePattern outside a route: got %q want empty", got)
	}
}
