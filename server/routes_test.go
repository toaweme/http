package server

import (
	"net/http"
	"testing"
)

// recordingRouter captures Handle calls so we can assert Register forwards
// every route without standing up a real router.
type recordingRouter struct {
	calls []Route
}

var _ HandleRouter = (*recordingRouter)(nil)

func (rr *recordingRouter) Handle(method, pattern string, handler http.Handler) {
	rr.calls = append(rr.calls, Route{Method: method, Pattern: pattern, Handler: handler})
}

func Test_Register_ForwardsEveryRoute(t *testing.T) {
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	routes := []Route{
		{Method: http.MethodGet, Pattern: "/a", Handler: h},
		{Method: http.MethodPost, Pattern: "/b", Handler: h},
		{Method: http.MethodDelete, Pattern: "/c/{id}", Handler: h},
	}

	rr := &recordingRouter{}
	Register(rr, routes)

	if len(rr.calls) != len(routes) {
		t.Fatalf("calls: got %d want %d", len(rr.calls), len(routes))
	}
	for i, want := range routes {
		got := rr.calls[i]
		if got.Method != want.Method || got.Pattern != want.Pattern {
			t.Fatalf("route %d: got %s %s want %s %s", i, got.Method, got.Pattern, want.Method, want.Pattern)
		}
	}
}

func Test_Register_EmptyIsNoop(t *testing.T) {
	rr := &recordingRouter{}
	Register(rr, nil)
	if len(rr.calls) != 0 {
		t.Fatalf("expected no calls, got %d", len(rr.calls))
	}
}
