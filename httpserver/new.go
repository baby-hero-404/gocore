package httpserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type serverImpl struct {
	engine     *gin.Engine
	httpServer *http.Server
	cfg        Config
}

// New initializes a Server with Gin engine and pre-wired standard middlewares.
func New(cfg Config, opts ...Option) Server {
	cfg.applyDefaults()

	engine := gin.New()

	// Register foundational middlewares by default
	engine.Use(RequestIDMiddleware())
	engine.Use(RecoveryMiddleware())
	engine.Use(RequestLoggerMiddleware())
	engine.Use(CORSMiddleware(cfg))

	srv := &serverImpl{
		engine: engine,
		cfg:    cfg,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      engine,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

// Engine exposes the underlying *gin.Engine for registering routes and middleware.
func (s *serverImpl) Engine() *gin.Engine {
	return s.engine
}
