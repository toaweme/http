package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func corsThrough(cfg CorsConfig, req *http.Request) (*httptest.ResponseRecorder, bool) {
	nextCalled := false
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	CrossOrigin(cfg)(h).ServeHTTP(rec, req)
	return rec, nextCalled
}

func Test_CrossOrigin(t *testing.T) {
	base := CorsConfig{
		AllowedOrigins:   []string{"http://localhost:4321", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           600,
	}

	tests := []struct {
		name        string
		cfg         CorsConfig
		method      string
		origin      string
		preflight   bool
		wantAllow   string
		wantCreds   string
		wantMethods string
		wantNext    bool
		wantStatus  int
	}{
		{
			name:       "no origin passes through untouched",
			cfg:        base,
			method:     http.MethodGet,
			origin:     "",
			wantAllow:  "",
			wantNext:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "allowed origin echoes back with credentials",
			cfg:        base,
			method:     http.MethodGet,
			origin:     "http://localhost:4321",
			wantAllow:  "http://localhost:4321",
			wantCreds:  "true",
			wantNext:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "disallowed origin gets no cors headers but still routes",
			cfg:        base,
			method:     http.MethodGet,
			origin:     "http://evil.example",
			wantAllow:  "",
			wantNext:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:        "preflight short-circuits with 204 and advertises policy",
			cfg:         base,
			method:      http.MethodOptions,
			origin:      "http://localhost:5174",
			preflight:   true,
			wantAllow:   "http://localhost:5174",
			wantCreds:   "true",
			wantMethods: "GET, POST, OPTIONS",
			wantNext:    false,
			wantStatus:  http.StatusNoContent,
		},
		{
			name:       "wildcard without credentials returns star",
			cfg:        CorsConfig{AllowedOrigins: []string{"*"}},
			method:     http.MethodGet,
			origin:     "http://anywhere.example",
			wantAllow:  "*",
			wantNext:   true,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/thing", http.NoBody)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.preflight {
				req.Header.Set("Access-Control-Request-Method", "POST")
			}

			rec, next := corsThrough(tt.cfg, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllow {
				t.Fatalf("allow-origin: got %q want %q", got, tt.wantAllow)
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != tt.wantCreds {
				t.Fatalf("allow-credentials: got %q want %q", got, tt.wantCreds)
			}
			if tt.wantMethods != "" {
				if got := rec.Header().Get("Access-Control-Allow-Methods"); got != tt.wantMethods {
					t.Fatalf("allow-methods: got %q want %q", got, tt.wantMethods)
				}
			}
			if next != tt.wantNext {
				t.Fatalf("next called: got %v want %v", next, tt.wantNext)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status: got %d want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
