package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, baseURL string, opts ...Option) Client {
	t.Helper()
	return NewClient(Config{BaseURL: baseURL}, opts...)
}

func Test_Client_Get(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("X-Custom", "yes")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	resp, err := client.Get(context.Background(), GetRequest{Request: Request{
		Path:  "/things",
		Query: url.Values{"a": {"1"}},
	}})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/things" {
		t.Errorf("path = %q, want /things", gotPath)
	}
	if gotQuery != "a=1" {
		t.Errorf("query = %q, want a=1", gotQuery)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if string(resp.Body) != "hello" {
		t.Errorf("body = %q, want hello", resp.Body)
	}
	if resp.Headers.Get("X-Custom") != "yes" {
		t.Errorf("X-Custom header = %q, want yes", resp.Headers.Get("X-Custom"))
	}
}

func Test_Client_Get_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("streamed-payload"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	resp, err := client.Get(context.Background(), GetRequest{Request: Request{Path: "/asset", Stream: true}})
	if err != nil {
		t.Fatalf("Get(Stream) returned error: %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	// a streamed response buffers nothing; the body arrives via the reader.
	if resp.Body != nil {
		t.Errorf("Body = %q, want nil for a streamed response", resp.Body)
	}
	if resp.Reader == nil {
		t.Fatal("Reader is nil for a streamed response")
	}
	// Response is an io.Reader, so io.Copy reads straight from it.
	var buf strings.Builder
	if _, err := io.Copy(&buf, resp); err != nil {
		t.Fatalf("io.Copy from response: %v", err)
	}
	if buf.String() != "streamed-payload" {
		t.Errorf("streamed body = %q, want streamed-payload", buf.String())
	}
}

func Test_Client_Get_BufferedClose(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("buffered"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	resp, err := client.Get(context.Background(), GetRequest{Request: Request{Path: "/x"}})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(resp.Body) != "buffered" {
		t.Errorf("Body = %q, want buffered", resp.Body)
	}
	if resp.Reader != nil {
		t.Errorf("Reader = %v, want nil for a buffered response", resp.Reader)
	}
	// Close is a no-op on a buffered response, so it is always safe to defer.
	if err := resp.Close(); err != nil {
		t.Errorf("Close on buffered response = %v, want nil", err)
	}
}

func Test_Client_BodyMethods(t *testing.T) {
	tests := []struct {
		name       string
		wantMethod string
		call       func(c Client, body []byte) (*Response, error)
	}{
		{
			name:       "post",
			wantMethod: http.MethodPost,
			call: func(c Client, body []byte) (*Response, error) {
				return c.Post(context.Background(), PostRequest{Request: Request{Path: "/p"}, Body: body})
			},
		},
		{
			name:       "put",
			wantMethod: http.MethodPut,
			call: func(c Client, body []byte) (*Response, error) {
				return c.Put(context.Background(), PutRequest{Request: Request{Path: "/p"}, Body: body})
			},
		},
		{
			name:       "patch",
			wantMethod: http.MethodPatch,
			call: func(c Client, body []byte) (*Response, error) {
				return c.Patch(context.Background(), PatchRequest{Request: Request{Path: "/p"}, Body: body})
			},
		},
		{
			name:       "delete",
			wantMethod: http.MethodDelete,
			call: func(c Client, body []byte) (*Response, error) {
				return c.Delete(context.Background(), Request{Path: "/p"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("ok"))
			}))
			defer srv.Close()

			client := newTestClient(t, srv.URL)
			resp, err := tt.call(client, []byte("payload"))
			if err != nil {
				t.Fatalf("%s returned error: %v", tt.name, err)
			}
			if gotMethod != tt.wantMethod {
				t.Errorf("method = %q, want %q", gotMethod, tt.wantMethod)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("status = %d, want 201", resp.StatusCode)
			}
			if tt.name == "delete" {
				if len(gotBody) != 0 {
					t.Errorf("delete sent body %q, want empty", gotBody)
				}
			} else if string(gotBody) != "payload" {
				t.Errorf("body = %q, want payload", gotBody)
			}
		})
	}
}

func Test_Client_SendsConfiguredHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(Config{
		BaseURL:     srv.URL,
		UserAgent:   "awee-cli/1.0.0",
		Platform:    "cli",
		AppVersion:  "1.2.3",
		ClientID:    "client-xyz",
		ServiceName: "svc",
		Headers:     map[string]string{"X-Extra": "extra"},
	})

	_, err := client.Get(context.Background(), GetRequest{Request: Request{
		Path:      "/x",
		ID:        "req-1",
		SessionID: "sess-1",
		Headers:   map[string]string{"X-Override": "ovr"},
	}})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	checks := map[string]string{
		ClientUserAgentHeaderName:  "awee-cli/1.0.0",
		ClientPlatformHeaderName:   "cli",
		ClientAppVersionHeaderName: "1.2.3",
		ClientIDHeaderName:         "client-xyz",
		ServiceNameHeaderName:      "svc",
		"X-Extra":                  "extra",
		"X-Override":               "ovr",
		ClientRequestIDHeaderName:  "req-1",
		ClientSessionIDHeaderName:  "sess-1",
	}
	for k, want := range checks {
		if got.Get(k) != want {
			t.Errorf("header %q = %q, want %q", k, got.Get(k), want)
		}
	}
}

func Test_Client_RequestHeaderOverridesConfig(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(Config{
		BaseURL: srv.URL,
		Headers: map[string]string{"X-Token": "config"},
	})
	_, err := client.Get(context.Background(), GetRequest{Request: Request{
		Path:    "/x",
		Headers: map[string]string{"X-Token": "request"},
	}})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Get("X-Token") != "request" {
		t.Errorf("X-Token = %q, want request (per-request override wins)", got.Get("X-Token"))
	}
}

func Test_Client_Get_RequestError(t *testing.T) {
	// invalid base URL scheme produces a transport error on Do.
	client := NewClient(Config{BaseURL: "http://no such host:invalid"})
	_, err := client.Get(context.Background(), GetRequest{Request: Request{Path: "/x"}})
	if err == nil {
		t.Fatal("expected error for invalid request, got nil")
	}
}

func Test_Client_PostStream_SSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Accept header = %q, want text/event-stream", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, ": a comment\n")
		_, _ = io.WriteString(w, "id: 42\n")
		_, _ = io.WriteString(w, "event: greeting\n")
		_, _ = io.WriteString(w, "retry: 1000\n")
		_, _ = io.WriteString(w, "data: hello\n")
		_, _ = io.WriteString(w, "data: [DONE]\n")
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	stream := make(chan StreamResponse)
	err := client.PostStream(context.Background(), stream, PostRequest{Request: Request{Path: "/sse"}})
	if err != nil {
		t.Fatalf("PostStream returned error: %v", err)
	}

	var types []StreamResponseType
	var bodies []string
	for msg := range stream {
		types = append(types, msg.Type)
		bodies = append(bodies, string(msg.Body))
	}

	wantTypes := []StreamResponseType{
		StreamResponseTypeComment,
		StreamResponseTypeID,
		StreamResponseTypeEvent,
		StreamResponseTypeRetry,
		StreamResponseTypeData,
		StreamResponseTypeEOF,
	}
	if !reflect.DeepEqual(types, wantTypes) {
		t.Errorf("stream types = %v, want %v", types, wantTypes)
	}
	wantBodies := []string{": a comment", "42", "greeting", "1000", "hello", ""}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Errorf("stream bodies = %v, want %v", bodies, wantBodies)
	}
}

func Test_Client_GetStream_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	stream := make(chan StreamResponse, 1)
	err := client.GetStream(context.Background(), stream, Request{Path: "/sse"})
	if err == nil {
		t.Fatal("expected error for non-200 stream, got nil")
	}

	msg, ok := <-stream
	if !ok {
		t.Fatal("expected an EOF message on the stream before close")
	}
	if msg.Type != StreamResponseTypeEOF {
		t.Errorf("type = %q, want EOF", msg.Type)
	}
	if msg.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", msg.StatusCode)
	}
	if string(msg.Body) != "boom" {
		t.Errorf("body = %q, want boom", msg.Body)
	}
	if _, ok := <-stream; ok {
		t.Error("stream should be closed after the error message")
	}
}

func Test_Client_WithLogger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rec := &recordingLogger{}
	client := NewClient(Config{BaseURL: srv.URL}, WithLogger(rec))
	if _, err := client.Get(context.Background(), GetRequest{Request: Request{Path: "/x"}}); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if rec.traces == 0 {
		t.Error("expected logger to receive trace calls, got none")
	}
}

func Test_Client_WithLogger_NilKeepsNopLogger(t *testing.T) {
	// passing a nil logger must not override the default nop logger, and must
	// not panic on use.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(Config{BaseURL: srv.URL}, WithLogger(nil))
	if _, err := client.Get(context.Background(), GetRequest{Request: Request{Path: "/x"}}); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

type stubRoundTripper struct {
	gotReq *http.Request
	resp   *http.Response
	err    error
}

func (s *stubRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	s.gotReq = r
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func Test_Client_WithHTTPClient_Injected(t *testing.T) {
	stub := &stubRoundTripper{
		resp: &http.Response{
			StatusCode: http.StatusTeapot,
			Body:       io.NopCloser(strings.NewReader("brewed")),
			Header:     http.Header{},
		},
	}
	client := NewClient(
		Config{BaseURL: "https://api.example.com"},
		WithHTTPClient(&http.Client{Transport: stub}),
	)

	resp, err := client.Get(context.Background(), GetRequest{Request: Request{Path: "/x"}})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if stub.gotReq == nil {
		t.Fatal("injected http client was not used")
	}
	if stub.gotReq.URL.String() != "https://api.example.com/x" {
		t.Errorf("request URL = %q, want https://api.example.com/x", stub.gotReq.URL.String())
	}
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("status = %d, want 418", resp.StatusCode)
	}
	if string(resp.Body) != "brewed" {
		t.Errorf("body = %q, want brewed", resp.Body)
	}
}

func Test_Client_WithHTTPClient_NilKeepsDefault(t *testing.T) {
	// a nil client must be ignored, leaving http.DefaultClient in place.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(Config{BaseURL: srv.URL}, WithHTTPClient(nil))
	if _, err := client.Get(context.Background(), GetRequest{Request: Request{Path: "/x"}}); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

func Test_BuildRequestParams(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		clientHdrs  map[string]string
		req         Request
		wantPath    string
		wantHeaders map[string]string
		wantErr     bool
	}{
		{
			name:     "base url joined with path",
			baseURL:  "https://api.example.com",
			req:      Request{Path: "/v1/things"},
			wantPath: "https://api.example.com/v1/things",
		},
		{
			name:     "no base url uses path as is",
			baseURL:  "",
			req:      Request{Path: "/v1/things"},
			wantPath: "/v1/things",
		},
		{
			name:     "query encoded and appended",
			baseURL:  "https://api.example.com",
			req:      Request{Path: "/search", Query: url.Values{"q": {"go lang"}, "n": {"5"}}},
			wantPath: "https://api.example.com/search?n=5&q=go+lang",
		},
		{
			name:       "client headers copied",
			baseURL:    "https://api.example.com",
			clientHdrs: map[string]string{"X-Base": "base"},
			req:        Request{Path: "/x"},
			wantPath:   "https://api.example.com/x",
			wantHeaders: map[string]string{
				"X-Base": "base",
			},
		},
		{
			name:       "request headers override client headers",
			baseURL:    "https://api.example.com",
			clientHdrs: map[string]string{"X-Base": "base"},
			req:        Request{Path: "/x", Headers: map[string]string{"X-Base": "req"}},
			wantPath:   "https://api.example.com/x",
			wantHeaders: map[string]string{
				"X-Base": "req",
			},
		},
		{
			name:     "id and session id become headers",
			baseURL:  "https://api.example.com",
			req:      Request{Path: "/x", ID: "req-1", SessionID: "sess-1"},
			wantPath: "https://api.example.com/x",
			wantHeaders: map[string]string{
				ClientRequestIDHeaderName: "req-1",
				ClientSessionIDHeaderName: "sess-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := httpClient{baseURL: tt.baseURL, headers: tt.clientHdrs}
			path, headers, err := h.buildRequestParams(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildRequestParams returned error: %v", err)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
			for k, want := range tt.wantHeaders {
				if headers[k] != want {
					t.Errorf("header %q = %q, want %q", k, headers[k], want)
				}
			}
		})
	}
}

func Test_BuildRequestParams_DoesNotMutateClientHeaders(t *testing.T) {
	clientHdrs := map[string]string{"X-Base": "base"}
	h := httpClient{baseURL: "https://api.example.com", headers: clientHdrs}
	_, headers, err := h.buildRequestParams(Request{Path: "/x", Headers: map[string]string{"X-Req": "req"}})
	if err != nil {
		t.Fatalf("buildRequestParams returned error: %v", err)
	}
	if _, ok := headers["X-Req"]; !ok {
		t.Error("returned headers should contain per-request header")
	}
	if _, ok := clientHdrs["X-Req"]; ok {
		t.Error("client headers must not be mutated by a per-request header")
	}
}

func Test_UserAgent(t *testing.T) {
	got := UserAgent("awee-cli", "1.0.0", "darwin", "23.0", "arm64")
	want := "awee-cli/1.0.0 (darwin 23.0; arm64)"
	if got != want {
		t.Errorf("UserAgent = %q, want %q", got, want)
	}
}

func Test_LimitBodySize(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		maxSize int64
		want    string
	}{
		{name: "under limit", body: "short", maxSize: 100, want: "short"},
		{name: "at limit", body: "abc", maxSize: 3, want: "abc"},
		{name: "over limit truncated", body: "abcdef", maxSize: 3, want: "abc..."},
		{name: "empty", body: "", maxSize: 10, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := limitBodySize([]byte(tt.body), tt.maxSize)
			if got != tt.want {
				t.Errorf("limitBodySize = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_LogArgs(t *testing.T) {
	base := []any{"a", 1}
	out := logArgs(base, "b", 2)
	want := []any{"a", 1, "b", 2}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("logArgs = %v, want %v", out, want)
	}
	// must not alias the base backing array.
	out[0] = "mutated"
	if base[0] != "a" {
		t.Error("logArgs must not mutate the base slice")
	}
}

type recordingLogger struct {
	traces int
	debugs int
	errs   int
}

var _ Logger = (*recordingLogger)(nil)

func (l *recordingLogger) Trace(string, ...any) { l.traces++ }
func (l *recordingLogger) Debug(string, ...any) { l.debugs++ }
func (l *recordingLogger) Info(string, ...any)  {}
func (l *recordingLogger) Warn(string, ...any)  {}
func (l *recordingLogger) Error(string, ...any) { l.errs++ }
