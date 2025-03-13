package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	
	"github.com/gin-gonic/gin"
	"github.com/zolia/go-ci/exithandler"
	
	"github.com/toaweme/log"
)

type Config struct {
	Host string
	Port int
}

type Handler interface {
	RegisterRoutes(router *gin.Engine)
}

type Server interface {
	Routes(auth gin.HandlerFunc, routes ...Handler)
}

var _ Server = (*GinServer)(nil)
var _ exithandler.Service = (*GinServer)(nil)

type GinServer struct {
	config *Config
	router *gin.Engine
	http   *http.Server
}

func NewGinServer(config *Config, router *gin.Engine) *GinServer {
	return &GinServer{
		config: config,
		router: router,
	}
}

func (g *GinServer) Name() string {
	return "http"
}

func (g *GinServer) Start() error {
	addr := fmt.Sprintf("%s:%d", g.config.Host, g.config.Port)
	g.http = &http.Server{
		Addr:    addr,
		Handler: g.router.Handler(),
	}
	
	log.Info("starting http server", "addr", fmt.Sprintf("http://%s", addr))
	if err := g.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("failed to start http server", "error", err)
		
		return fmt.Errorf("failed to start http server: %w", err)
	}
	
	return nil
}

func (g *GinServer) Stop(ctx context.Context) error {
	err := g.http.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}
	
	return nil
}

func (g *GinServer) Routes(auth gin.HandlerFunc, routes ...Handler) {
	for _, route := range routes {
		route.RegisterRoutes(g.router)
	}
}
