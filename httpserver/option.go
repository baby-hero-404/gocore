package httpserver

import "github.com/gin-gonic/gin"

// Option configures Server initialization.
type Option func(*serverImpl)

// WithMiddlewares appends additional Gin middlewares.
func WithMiddlewares(m ...gin.HandlerFunc) Option {
	return func(s *serverImpl) {
		s.engine.Use(m...)
	}
}
