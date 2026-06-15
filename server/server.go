package server

import (
	"context"
	"errors"
	"fmt"
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

// Option mutates the underlying *http.Server during construction. Options run
// after the defaults (Addr, Handler, ReadHeaderTimeout) are applied, so they
// can override anything.
type Option func(*http.Server)

// WithReadHeaderTimeout sets how long the server waits for request headers.
// Pass 0 to disable it (not recommended - exposes the server to Slowloris).
func WithReadHeaderTimeout(d time.Duration) Option {
	return func(srv *http.Server) { srv.ReadHeaderTimeout = d }
}

// WithReadTimeout sets the maximum duration for reading an entire request.
func WithReadTimeout(d time.Duration) Option {
	return func(srv *http.Server) { srv.ReadTimeout = d }
}

// WithWriteTimeout sets the maximum duration before timing out response writes.
func WithWriteTimeout(d time.Duration) Option {
	return func(srv *http.Server) { srv.WriteTimeout = d }
}

// WithIdleTimeout sets how long to keep a keep-alive connection idle.
func WithIdleTimeout(d time.Duration) Option {
	return func(srv *http.Server) { srv.IdleTimeout = d }
}

// Server wraps net/http.Server around a *Router. Implements the
// {Name, Start, Stop} contract expected by go-shared/service.Service.
type Server struct {
	config Config
	router *Router
	logger Logger
	http   *http.Server
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
	for _, opt := range opts {
		opt(srv)
	}
	return &Server{config: cfg, router: router, logger: logger, http: srv}
}

// HTTP returns the underlying *http.Server for callers that need to set fields
// no Option covers (TLS config, connection state hooks, error log, ...).
// Mutate it before calling Start; changes after the server is serving have no
// effect.
func (s *Server) HTTP() *http.Server { return s.http }

// Name identifies the service in a service registry.
func (s *Server) Name() string { return "http" }

// Start serves until Stop is called. It blocks and returns nil on a clean
// shutdown.
func (s *Server) Start() error {
	s.router.LogRoutes(s.logger)

	s.logger.Info("service", "http", "server", "addr", "http://"+s.http.Addr)
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("service", "http", "server", "error", err)
		return err
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
