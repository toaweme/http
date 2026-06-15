package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Config configures a Server's listen address.
type Config struct {
	Host string
	Port int
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
// can be injected directly, or a null logger to discard output.
func NewServer(cfg Config, router *Router, logger Logger) *Server {
	return &Server{config: cfg, router: router, logger: logger}
}

func (s *Server) Name() string { return "http" }

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.http = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	s.router.LogRoutes(s.logger)

	s.logger.Info("service", "http", "server", "addr", fmt.Sprintf("http://%s", addr))
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("service", "http", "server", "error", err)
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}
	return nil
}
