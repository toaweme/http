package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureLogger records the args of the last Info call so tests can assert
// what the middleware logged.
type captureLogger struct {
	last map[string]any
}

var _ Logger = (*captureLogger)(nil)

func (c *captureLogger) Trace(string, ...any) {}
func (c *captureLogger) Debug(string, ...any) {}
func (c *captureLogger) Warn(string, ...any)  {}
func (c *captureLogger) Error(string, ...any) {}
func (c *captureLogger) Info(_ string, args ...any) {
	c.last = make(map[string]any, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		key, _ := args[i].(string)
		c.last[key] = args[i+1]
	}
}

func serveThrough(cfg SlogConfig, logger Logger, h http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	mw := SlogMiddleware(cfg, logger)
	rec := httptest.NewRecorder()
	mw(h).ServeHTTP(rec, req)
	return rec
}

func Test_SlogMiddleware_LogsBasics(t *testing.T) {
	log := &captureLogger{}
	req := httptest.NewRequest(http.MethodGet, "/items?q=1", http.NoBody)
	serveThrough(SlogConfig{}, log, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}, req)

	if log.last["method"] != http.MethodGet {
		t.Fatalf("method: got %v", log.last["method"])
	}
	if log.last["url"] != "/items?q=1" {
		t.Fatalf("url: got %v", log.last["url"])
	}
	if log.last["code"] != http.StatusTeapot {
		t.Fatalf("code: got %v want %d", log.last["code"], http.StatusTeapot)
	}
	if _, ok := log.last["duration"]; !ok {
		t.Fatal("duration not logged")
	}
}

func Test_SlogMiddleware_DefaultStatusWhenHandlerSilent(t *testing.T) {
	log := &captureLogger{}
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	serveThrough(SlogConfig{}, log, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok")) // Write without explicit WriteHeader
	}, req)

	if log.last["code"] != http.StatusOK {
		t.Fatalf("code: got %v want 200", log.last["code"])
	}
}

func Test_SlogMiddleware_CapturesRequestBodyAndPreservesItDownstream(t *testing.T) {
	log := &captureLogger{}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))

	var seenByHandler string
	serveThrough(SlogConfig{LogRequestBody: true, MaxBodyBytes: 4}, log, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seenByHandler = string(b)
		w.WriteHeader(http.StatusOK)
	}, req)

	if got := log.last["request-body"]; got != "hell" {
		t.Fatalf("logged request-body: got %v want truncated 'hell'", got)
	}
	if seenByHandler != "hello" {
		t.Fatalf("handler saw %q want full 'hello'", seenByHandler)
	}
}

func Test_SlogMiddleware_CapturesResponseBodyWithCap(t *testing.T) {
	log := &captureLogger{}
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := serveThrough(SlogConfig{LogResponseBody: true, MaxBodyBytes: 5}, log, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}, req)

	if got := log.last["response-body"]; got != "01234" {
		t.Fatalf("logged response-body: got %v want capped '01234'", got)
	}
	if rec.Body.String() != "0123456789" {
		t.Fatalf("client saw %q want full body", rec.Body.String())
	}
}

func Test_SlogMiddleware_LogsHeadersWhenEnabled(t *testing.T) {
	log := &captureLogger{}
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Trace", "abc")
	serveThrough(SlogConfig{LogRequestHeaders: true, LogResponseHeaders: true}, log, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Out", "def")
		w.WriteHeader(http.StatusOK)
	}, req)

	reqHeaders, ok := log.last["request-headers"].(map[string]string)
	if !ok || reqHeaders["X-Trace"] != "abc" {
		t.Fatalf("request-headers: got %v", log.last["request-headers"])
	}
	respHeaders, ok := log.last["response-headers"].(map[string]string)
	if !ok || respHeaders["X-Out"] != "def" {
		t.Fatalf("response-headers: got %v", log.last["response-headers"])
	}
}

func Test_responseRecorder_DoubleWriteHeaderIgnored(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := &responseRecorder{ResponseWriter: rec, status: http.StatusOK}
	rr.WriteHeader(http.StatusCreated)
	rr.WriteHeader(http.StatusInternalServerError) // must be ignored

	if rr.status != http.StatusCreated {
		t.Fatalf("status: got %d want %d (first WriteHeader wins)", rr.status, http.StatusCreated)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("underlying status: got %d want %d", rec.Code, http.StatusCreated)
	}
}

func Test_flattenHeaders_JoinsMultiValue(t *testing.T) {
	h := http.Header{}
	h.Add("X-Multi", "a")
	h.Add("X-Multi", "b")
	h.Set("X-Single", "one")

	out := flattenHeaders(h)
	if out["X-Multi"] != "a, b" {
		t.Fatalf("multi: got %q want 'a, b'", out["X-Multi"])
	}
	if out["X-Single"] != "one" {
		t.Fatalf("single: got %q want 'one'", out["X-Single"])
	}
}
