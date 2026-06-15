package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func Test_JSONEvent(t *testing.T) {
	t.Run("marshals payload into data", func(t *testing.T) {
		ev, err := JSONEvent("update", map[string]int{"n": 1})
		if err != nil {
			t.Fatalf("JSONEvent: %v", err)
		}
		if ev.Type != "update" {
			t.Fatalf("type: got %q want update", ev.Type)
		}
		if ev.Data != `{"n":1}` {
			t.Fatalf("data: got %q", ev.Data)
		}
	})

	t.Run("errors on unmarshalable payload", func(t *testing.T) {
		if _, err := JSONEvent("x", make(chan int)); err == nil {
			t.Fatal("expected error marshaling a channel, got nil")
		}
	})
}

func Test_NewWriter_RequiresFlusher(t *testing.T) {
	if _, err := NewWriter(nonFlusher{httptest.NewRecorder()}); err == nil {
		t.Fatal("expected error for non-flushing writer, got nil")
	}
	if _, err := NewWriter(httptest.NewRecorder()); err != nil {
		t.Fatalf("flusher recorder should succeed: %v", err)
	}
}

// nonFlusher wraps a ResponseWriter while hiding any Flush method.
type nonFlusher struct{ http.ResponseWriter }

func Test_Writer_StartSetsHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	w.Start()
	w.Start() // idempotent

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type: got %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("cache-control: got %q", cc)
	}
	if rec.Header().Get("X-Accel-Buffering") != "no" {
		t.Fatal("X-Accel-Buffering not set")
	}
}

func Test_Writer_WriteFormat(t *testing.T) {
	tests := []struct {
		name string
		ev   Event
		want string
	}{
		{
			name: "full event",
			ev:   Event{ID: "7", Type: "msg", Data: "hello"},
			want: "id: 7\nevent: msg\ndata: hello\n\n",
		},
		{
			name: "default message event omits id and event lines",
			ev:   Event{Data: "x"},
			want: "data: x\n\n",
		},
		{
			name: "multi-line data emits one data line per line",
			ev:   Event{Data: "a\nb"},
			want: "data: a\ndata: b\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			w, err := NewWriter(rec)
			if err != nil {
				t.Fatalf("NewWriter: %v", err)
			}
			if err := w.Write(tt.ev); err != nil {
				t.Fatalf("Write: %v", err)
			}
			// strip the streaming headers preamble (none in body); body is the event
			if !strings.HasSuffix(rec.Body.String(), tt.want) {
				t.Fatalf("body: got %q want suffix %q", rec.Body.String(), tt.want)
			}
		})
	}
}

func Test_Writer_Ping(t *testing.T) {
	rec := httptest.NewRecorder()
	w, err := NewWriter(rec)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if !strings.HasSuffix(rec.Body.String(), ": ping\n\n") {
		t.Fatalf("ping body: got %q", rec.Body.String())
	}
}

func Test_Hub_PublishDelivers(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.Subscribe(t.Context(), "topic", 8)
	defer cancel()

	hub.Publish("topic", Event{Type: "x", Data: "payload"})

	select {
	case ev := <-ch:
		if ev.Data != "payload" {
			t.Fatalf("data: got %q want payload", ev.Data)
		}
		if ev.ID == "" {
			t.Fatal("expected auto-assigned ID for event without one")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func Test_Hub_PreservesProvidedID(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.Subscribe(t.Context(), "t", 8)
	defer cancel()

	hub.Publish("t", Event{ID: "custom", Data: "x"})
	if ev := <-ch; ev.ID != "custom" {
		t.Fatalf("ID: got %q want custom", ev.ID)
	}
}

func Test_Hub_Subscribers(t *testing.T) {
	hub := NewHub()
	if n := hub.Subscribers("t"); n != 0 {
		t.Fatalf("initial subscribers: got %d want 0", n)
	}
	_, cancel := hub.Subscribe(t.Context(), "t", 8)
	if n := hub.Subscribers("t"); n != 1 {
		t.Fatalf("after subscribe: got %d want 1", n)
	}
	cancel()
	if n := hub.Subscribers("t"); n != 0 {
		t.Fatalf("after cancel: got %d want 0", n)
	}
}

func Test_Hub_PublishToNoSubscribersIsNoop(t *testing.T) {
	hub := NewHub()
	hub.Publish("empty", Event{Data: "x"}) // must not panic
}

func Test_Hub_DropsSlowSubscriber(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.Subscribe(t.Context(), "t", 1)
	defer cancel()

	hub.Publish("t", Event{Data: "1"}) // fills the buffer
	hub.Publish("t", Event{Data: "2"}) // buffer full -> subscriber dropped

	// first event is buffered and readable
	if ev := <-ch; ev.Data != "1" {
		t.Fatalf("first event: got %q want 1", ev.Data)
	}
	// channel is now closed because the subscriber was dropped
	if _, ok := <-ch; ok {
		t.Fatal("expected channel closed after slow-subscriber drop")
	}
	if n := hub.Subscribers("t"); n != 0 {
		t.Fatalf("dropped subscriber still counted: got %d want 0", n)
	}
}

func Test_Hub_ContextCancelUnsubscribes(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(t.Context())
	hub.Subscribe(ctx, "t", 8)
	cancel()

	deadline := time.Now().Add(time.Second)
	for hub.Subscribers("t") != 0 {
		if time.Now().After(deadline) {
			t.Fatal("subscriber not removed after context cancel")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func Test_ServeStream(t *testing.T) {
	hub := NewHub()
	rec := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(t.Context())
	req := httptest.NewRequest(http.MethodGet, "/stream", http.NoBody).WithContext(ctx)

	done := make(chan error, 1)
	go func() { done <- ServeStream(rec, req, hub, "topic") }()

	// wait for ServeStream to subscribe before publishing
	deadline := time.Now().Add(time.Second)
	for hub.Subscribers("topic") == 0 {
		if time.Now().After(deadline) {
			t.Fatal("ServeStream never subscribed")
		}
		time.Sleep(5 * time.Millisecond)
	}

	hub.Publish("topic", Event{Type: "msg", Data: "live"})
	time.Sleep(20 * time.Millisecond)
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("ServeStream: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "data: live") {
		t.Fatalf("stream body missing published event: %q", rec.Body.String())
	}
}
