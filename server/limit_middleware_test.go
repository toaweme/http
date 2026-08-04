package server

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_LimitBody(t *testing.T) {
	tests := []struct {
		name     string
		limit    int64
		override int64
		body     string
		wantRead int
		wantErr  bool
	}{
		{name: "under the limit", limit: 16, body: "hello", wantRead: 5},
		{name: "at the limit", limit: 5, body: "hello", wantRead: 5},
		{name: "over the limit", limit: 4, body: "hello", wantRead: 4, wantErr: true},
		{name: "no limit reads everything", limit: 0, body: strings.Repeat("x", 512), wantRead: 512},
		{name: "override widens the router limit", limit: 4, override: 64, body: "hello", wantRead: 5},
		{name: "override narrows the router limit", limit: 64, override: 2, body: "hello", wantRead: 2, wantErr: true},
		{name: "override of zero keeps the router limit", limit: 4, override: 0, body: "hello", wantRead: 4, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var read int
			var readErr error
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.override != 0 {
					r = LimitRequestBody(w, r, tt.override)
				}
				body, err := io.ReadAll(r.Body)
				read, readErr = len(body), err
			})

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			LimitBody(tt.limit)(handler).ServeHTTP(httptest.NewRecorder(), req)

			if read != tt.wantRead {
				t.Fatalf("read %d bytes, want %d", read, tt.wantRead)
			}
			var tooLarge *http.MaxBytesError
			if got := errors.As(readErr, &tooLarge); got != tt.wantErr {
				t.Fatalf("too large error %v, want %v (err %v)", got, tt.wantErr, readErr)
			}
		})
	}
}

// Test_LimitBody_NilBody covers a request built without a body, which a handler
// invoked directly rather than by the server can still carry.
func Test_LimitBody_NilBody(t *testing.T) {
	served := false
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { served = true })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Body = nil
	LimitBody(8)(handler).ServeHTTP(httptest.NewRecorder(), req)

	if !served {
		t.Fatal("handler was not reached")
	}
}
