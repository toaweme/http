package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	r := NewRouter()
	r.Use(AuthMiddleware(func(string) (*Claims, error) { return &Claims{OrgID: "o", UserID: "u"}, nil }))
	Register(r, []Route{{Method: "GET", Pattern: "/x", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing header: want 401 got %d", w.Code)
	}
}

func TestAuthMiddleware_ExtractorError(t *testing.T) {
	r := NewRouter()
	r.Use(AuthMiddleware(func(string) (*Claims, error) { return nil, errors.New("bad") }))
	Register(r, []Route{{Method: "GET", Pattern: "/x", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("handler should not be called")
	})}})

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("extractor error: want 401 got %d", w.Code)
	}
}

func TestAuthMiddleware_InjectsContext(t *testing.T) {
	r := NewRouter()
	r.Use(AuthMiddleware(func(token string) (*Claims, error) {
		if token != "abc" {
			return nil, errors.New("nope")
		}
		return &Claims{OrgID: "org-1", UserID: "user-1", Scopes: []string{"read"}}, nil
	}))
	var gotOrg, gotUser string
	var gotScopes []string
	Register(r, []Route{{Method: "GET", Pattern: "/x", Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotOrg, _ = OrgIDFromContext(req.Context())
		gotUser, _ = UserIDFromContext(req.Context())
		gotScopes = ScopesFromContext(req.Context())
		w.WriteHeader(http.StatusOK)
	})}})

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", w.Code)
	}
	if gotOrg != "org-1" || gotUser != "user-1" || len(gotScopes) != 1 || gotScopes[0] != "read" {
		t.Fatalf("ctx mismatch: org=%q user=%q scopes=%v", gotOrg, gotUser, gotScopes)
	}
}

func TestRouter_PathParams(t *testing.T) {
	r := NewRouter()
	var gotID string
	Register(r, []Route{{Method: "GET", Pattern: "/items/{id}", Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotID = req.PathValue("id")
		w.WriteHeader(http.StatusOK)
	})}})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/items/42", nil))
	if w.Code != http.StatusOK || gotID != "42" {
		t.Fatalf("want 200 id=42 got code=%d id=%q", w.Code, gotID)
	}
}
