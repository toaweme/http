// Package sse provides a small Server-Sent Events writer and broadcast Hub.
//
// Layout:
//
//   - Writer: a thin wrapper around an http.ResponseWriter that knows how to
//     emit single SSE events (id, event, data) and flush the connection.
//   - Hub: subscribers/topics. A producer calls Publish; every subscriber on
//     that topic receives the event via its channel. Subscribers that fall
//     behind get their channel closed rather than blocking the producer.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Event is one SSE record. ID is optional; Type maps to "event:"; Data is the
// payload. Empty Type emits a default "message" event.
type Event struct {
	ID   string
	Type string
	Data string
}

// JSONEvent is a convenience constructor - marshals the payload into Data.
// Returns an error if marshalling fails; otherwise the event is ready to
// publish or write.
func JSONEvent(eventType string, payload any) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("failed to marshal sse payload: %w", err)
	}
	return Event{Type: eventType, Data: string(raw)}, nil
}

// Writer wraps an http.ResponseWriter and tracks the underlying Flusher so
// every Write reaches the client immediately. The first call sets the
// streaming headers; subsequent calls just emit events.
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
	started bool
}

// NewWriter constructs a Writer. Returns nil, error if the response writer
// does not support flushing - SSE is impossible without it.
func NewWriter(w http.ResponseWriter) (*Writer, error) {
	f, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("response writer does not support flushing - SSE requires http.Flusher")
	}
	return &Writer{w: w, flusher: f}, nil
}

// Start writes the streaming headers and flushes. Safe to call multiple times.
func (w *Writer) Start() {
	if w.started {
		return
	}
	h := w.w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	// disable response buffering proxies - nginx in particular needs this
	h.Set("X-Accel-Buffering", "no")
	w.w.WriteHeader(http.StatusOK)
	w.flusher.Flush()
	w.started = true
}

// Write emits one SSE event and flushes. Returns the underlying write error
// (typically because the client disconnected) so the caller knows to stop.
func (w *Writer) Write(ev Event) error {
	if !w.started {
		w.Start()
	}
	var b strings.Builder
	if ev.ID != "" {
		b.WriteString("id: ")
		b.WriteString(ev.ID)
		b.WriteByte('\n')
	}
	if ev.Type != "" {
		b.WriteString("event: ")
		b.WriteString(ev.Type)
		b.WriteByte('\n')
	}
	// data may be multi-line - emit one `data:` per line per spec
	for _, line := range strings.Split(ev.Data, "\n") {
		b.WriteString("data: ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	if _, err := w.w.Write([]byte(b.String())); err != nil {
		return fmt.Errorf("failed to write sse event: %w", err)
	}
	w.flusher.Flush()
	return nil
}

// Ping emits an SSE comment line - useful as a heartbeat to keep proxies
// from idle-closing the connection. Comments are ignored by clients.
func (w *Writer) Ping() error {
	if !w.started {
		w.Start()
	}
	if _, err := w.w.Write([]byte(": ping\n\n")); err != nil {
		return fmt.Errorf("failed to write sse ping: %w", err)
	}
	w.flusher.Flush()
	return nil
}

// Hub is a topic-scoped fanout. Each topic has zero or more subscribers, each
// holding a buffered channel. Producers Publish to a topic; subscribers
// receive. Slow subscribers get dropped - their channel is closed and they're
// removed from the topic.
type Hub struct {
	mu     sync.RWMutex
	topics map[string]map[*subscription]struct{}

	// idSeq generates monotonic IDs for events that don't carry their own.
	idSeq atomic.Uint64
}

// NewHub builds an empty hub.
func NewHub() *Hub {
	return &Hub{topics: make(map[string]map[*subscription]struct{})}
}

type subscription struct {
	ch chan Event
}

// Subscribe registers a new subscriber on the given topic. The returned
// channel receives every event published to the topic; cancel via ctx (or
// call the returned cancel func) to unsubscribe and free the channel.
//
// Buffer size controls how many events can pile up before the subscriber is
// considered slow and dropped.
func (h *Hub) Subscribe(ctx context.Context, topic string, buffer int) (<-chan Event, func()) {
	if buffer <= 0 {
		buffer = 64
	}
	sub := &subscription{ch: make(chan Event, buffer)}

	h.mu.Lock()
	if h.topics[topic] == nil {
		h.topics[topic] = make(map[*subscription]struct{})
	}
	h.topics[topic][sub] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if subs, ok := h.topics[topic]; ok {
			if _, present := subs[sub]; present {
				delete(subs, sub)
				close(sub.ch)
			}
			if len(subs) == 0 {
				delete(h.topics, topic)
			}
		}
		h.mu.Unlock()
	}

	if ctx != nil {
		go func() {
			<-ctx.Done()
			cancel()
		}()
	}
	return sub.ch, cancel
}

// Publish sends an event to every subscriber of the topic. Subscribers whose
// channels are full at the moment of delivery are considered slow and have
// their channels closed (the subscriber sees a closed-channel receive and
// can decide to reconnect).
func (h *Hub) Publish(topic string, ev Event) {
	if ev.ID == "" {
		ev.ID = strconv.FormatUint(h.idSeq.Add(1), 10)
	}
	h.mu.RLock()
	subs := h.topics[topic]
	if len(subs) == 0 {
		h.mu.RUnlock()
		return
	}
	// copy to avoid holding the lock while sending
	pending := make([]*subscription, 0, len(subs))
	for s := range subs {
		pending = append(pending, s)
	}
	h.mu.RUnlock()

	var slow []*subscription
	for _, s := range pending {
		select {
		case s.ch <- ev:
		default:
			slow = append(slow, s)
		}
	}
	if len(slow) > 0 {
		h.mu.Lock()
		subSet := h.topics[topic]
		for _, s := range slow {
			if subSet != nil {
				if _, present := subSet[s]; present {
					delete(subSet, s)
					close(s.ch)
				}
			}
		}
		if subSet != nil && len(subSet) == 0 {
			delete(h.topics, topic)
		}
		h.mu.Unlock()
	}
}

// Subscribers returns the current subscriber count for the topic. Useful for
// metrics and for deciding whether to keep producing events.
func (h *Hub) Subscribers(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.topics[topic])
}

// ServeStream is a convenience handler that subscribes the client to a topic
// and streams every event over SSE until the request context is cancelled.
// Sends a heartbeat ping every 15s so proxies don't idle out the connection.
func ServeStream(w http.ResponseWriter, r *http.Request, hub *Hub, topic string) error {
	sw, err := NewWriter(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	sw.Start()
	ch, cancel := hub.Subscribe(r.Context(), topic, 256)
	defer cancel()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-heartbeat.C:
			if err := sw.Ping(); err != nil {
				return nil
			}
		case ev, ok := <-ch:
			if !ok {
				// hub dropped us - tell the client to reconnect via a final
				// event so it knows the stream ended deliberately.
				_ = sw.Write(Event{Type: "sse.dropped", Data: "subscriber fell behind"})
				return nil
			}
			if err := sw.Write(ev); err != nil {
				return nil
			}
		}
	}
}
