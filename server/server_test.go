package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func Test_NewServer_Defaults(t *testing.T) {
	r := NewRouter()
	s := NewServer(Config{Host: "127.0.0.1", Port: 8080}, r, nopLogger{})

	if s.Name() != "http" {
		t.Fatalf("Name: got %q want http", s.Name())
	}
	srv := s.HTTP()
	if srv == nil {
		t.Fatal("HTTP() returned nil")
	}
	if srv.Addr != "127.0.0.1:8080" {
		t.Fatalf("Addr: got %q want 127.0.0.1:8080", srv.Addr)
	}
	if srv.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout: got %v want %v", srv.ReadHeaderTimeout, defaultReadHeaderTimeout)
	}
	if srv.Handler != r {
		t.Fatal("Handler is not the router passed to NewServer")
	}
}

func Test_NewServer_OptionsApplied(t *testing.T) {
	r := NewRouter()
	s := NewServer(Config{Host: "", Port: 0}, r, nopLogger{},
		WithReadHeaderTimeout(3*time.Second),
		WithReadTimeout(4*time.Second),
		WithWriteTimeout(5*time.Second),
		WithIdleTimeout(6*time.Second),
	)

	srv := s.HTTP()
	if srv.ReadHeaderTimeout != 3*time.Second {
		t.Fatalf("ReadHeaderTimeout: got %v want 3s", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != 4*time.Second {
		t.Fatalf("ReadTimeout: got %v want 4s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 5*time.Second {
		t.Fatalf("WriteTimeout: got %v want 5s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 6*time.Second {
		t.Fatalf("IdleTimeout: got %v want 6s", srv.IdleTimeout)
	}
}

func Test_NewServer_OptionsOverrideDefault(t *testing.T) {
	s := NewServer(Config{}, NewRouter(), nopLogger{}, WithReadHeaderTimeout(0))
	if got := s.HTTP().ReadHeaderTimeout; got != 0 {
		t.Fatalf("ReadHeaderTimeout: got %v want 0 (disabled via option)", got)
	}
}

func Test_Server_HTTPEscapeHatch(t *testing.T) {
	s := NewServer(Config{}, NewRouter(), nopLogger{})
	// fields no Option covers are still reachable before Start
	s.HTTP().MaxHeaderBytes = 4096
	if s.HTTP().MaxHeaderBytes != 4096 {
		t.Fatalf("MaxHeaderBytes mutation via HTTP() did not stick")
	}
}

func Test_Server_StopBeforeStart(t *testing.T) {
	s := NewServer(Config{}, NewRouter(), nopLogger{})
	if err := s.Stop(t.Context()); err != nil {
		t.Fatalf("Stop before Start: got %v want nil", err)
	}
}

func Test_Server_StartStopLifecycle(t *testing.T) {
	port := freePort(t)
	r := NewRouter()
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})
	s := NewServer(Config{Host: "127.0.0.1", Port: port}, r, nopLogger{})

	errCh := make(chan error, 1)
	go func() { errCh <- s.Start() }()

	url := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	waitReachable(t, url)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := s.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Start returned error after clean shutdown: %v", err)
	}
}

// freePort binds an ephemeral port, releases it, and returns the number. There
// is an inherent race before the server re-binds, but it is acceptable in tests.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitReachable(t *testing.T, url string) {
	t.Helper()
	for range 100 {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server never became reachable at %s", url)
}
