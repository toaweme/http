package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func Benchmark_BuildRequestParams(b *testing.B) {
	h := httpClient{
		baseURL: "https://api.example.com",
		headers: map[string]string{"X-Base": "base", "X-Other": "other"},
	}
	req := Request{
		Path:      "/v1/things",
		ID:        "req-1",
		SessionID: "sess-1",
		Query:     url.Values{"q": {"go lang"}, "n": {"5"}},
		Headers:   map[string]string{"X-Override": "ovr"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := h.buildRequestParams(req); err != nil {
			b.Fatalf("buildRequestParams returned error: %v", err)
		}
	}
}

func Benchmark_Client_Get(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	client := NewClient(Config{BaseURL: srv.URL})
	req := GetRequest{Request: Request{Path: "/things", Query: url.Values{"a": {"1"}}}}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Get(ctx, req); err != nil {
			b.Fatalf("Get returned error: %v", err)
		}
	}
}

func Benchmark_Client_Post(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(Config{BaseURL: srv.URL})
	req := PostRequest{Request: Request{Path: "/things"}, Body: []byte(`{"name":"thing","value":42}`)}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Post(ctx, req); err != nil {
			b.Fatalf("Post returned error: %v", err)
		}
	}
}

func Benchmark_Client_PostStream_SSE(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 10; i++ {
			_, _ = io.WriteString(w, "event: tick\n")
			_, _ = io.WriteString(w, "data: payload\n")
		}
		_, _ = io.WriteString(w, "data: [DONE]\n")
	}))
	defer srv.Close()

	client := NewClient(Config{BaseURL: srv.URL})
	req := PostRequest{Request: Request{Path: "/sse"}}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream := make(chan StreamResponse)
		if err := client.PostStream(ctx, stream, req); err != nil {
			b.Fatalf("PostStream returned error: %v", err)
		}
		var got int
		for range stream {
			got++
		}
		_ = got
	}
}

func Benchmark_JSON(b *testing.B) {
	data := map[string]any{"name": "thing", "value": 42, "tags": []string{"a", "b", "c"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := JSON(data); err != nil {
			b.Fatalf("JSON returned error: %v", err)
		}
	}
}

func Benchmark_FromJSON(b *testing.B) {
	raw := []byte(`{"name":"thing","value":42,"tags":["a","b","c"]}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := FromJSON[map[string]any](raw); err != nil {
			b.Fatalf("FromJSON returned error: %v", err)
		}
	}
}

func Benchmark_LimitBodySize(b *testing.B) {
	body := make([]byte, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limitBodySize(body, size)
	}
}

func Benchmark_LogArgs(b *testing.B) {
	base := []any{"type", "stream-request", "method", "GET", "url", "https://api.example.com/x"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logArgs(base, "status", 200)
	}
}
