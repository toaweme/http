package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// defaultReadHeaderTimeout bounds how long the server waits for request
// headers, guarding against Slowloris-style connections. Applied at
// construction; override with WithReadHeaderTimeout or by mutating HTTP().
const defaultReadHeaderTimeout = 10 * time.Second

// Config configures a Server's listen address. Anything beyond the address
// (timeouts, TLS, connection hooks) is set via Option or by mutating the
// underlying server returned by HTTP.
type Config struct {
	Host string
	Port int
}

// Option mutates the Server during construction. Options run
// after the defaults (Addr, Handler, ReadHeaderTimeout) are applied, so they
// can override anything.
type Option func(*Server)

// WithReadHeaderTimeout sets how long the server waits for request headers.
// Pass 0 to disable it (not recommended - exposes the server to Slowloris).
func WithReadHeaderTimeout(d time.Duration) Option {
	return func(s *Server) { s.http.ReadHeaderTimeout = d }
}

// WithReadTimeout sets the maximum duration for reading an entire request.
func WithReadTimeout(d time.Duration) Option {
	return func(s *Server) { s.http.ReadTimeout = d }
}

// WithWriteTimeout sets the maximum duration before timing out response writes.
func WithWriteTimeout(d time.Duration) Option {
	return func(s *Server) { s.http.WriteTimeout = d }
}

// WithIdleTimeout sets how long to keep a keep-alive connection idle.
func WithIdleTimeout(d time.Duration) Option {
	return func(s *Server) { s.http.IdleTimeout = d }
}

// WithListener serves on a listener the caller opened,
// such as one bound to port 0 whose chosen port the caller needed before serving.
// Config's host and port are then ignored.
func WithListener(l net.Listener) Option {
	return func(s *Server) { s.listener = l }
}

// Server wraps net/http.Server around a *Router. Implements the
// {Name, Start, Stop} contract expected by go-shared/service.Service.
type Server struct {
	config   Config
	router   *Router
	logger   Logger
	http     *http.Server
	listener net.Listener
}

// NewServer wires a Server around the router. A github.com/toaweme/log logger
// can be injected directly, or a null logger to discard output. Pass Options
// to tune the underlying *http.Server, or reach for HTTP to set fields no
// Option covers.
func NewServer(cfg Config, router *Router, logger Logger, opts ...Option) *Server {
	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}
	s := &Server{config: cfg, router: router, logger: logger, http: srv}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// HTTP returns the underlying *http.Server for callers that need to set fields
// no Option covers (TLS config, connection state hooks, error log, ...).
// Mutate it before calling Start; changes after the server is serving have no
// effect.
func (s *Server) HTTP() *http.Server { return s.http }

// Name identifies the service in a service registry.
func (s *Server) Name() string { return "http" }

// Start serves until Stop is called, on the listener given with WithListener
// or else on Config's address. It blocks and returns nil on a clean shutdown.
func (s *Server) Start() error {
	s.router.LogRoutes(s.logger)

	addr := s.http.Addr
	serve := s.http.ListenAndServe
	if s.listener != nil {
		addr = s.listener.Addr().String()
		serve = func() error { return s.http.Serve(s.listener) }
	}

	s.logger.Info("service", "http", "server", "addr", "http://"+addr)
	if err := serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("service", "http", "server", "error", err)
		return fmt.Errorf("failed to serve on %s: %w", addr, err)
	}
	return nil
}

// Stop gracefully shuts the server down, respecting ctx's deadline.
func (s *Server) Stop(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}
	return nil
}
