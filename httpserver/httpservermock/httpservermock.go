// Package httpservermock provides test doubles for github.com/baby-hero-404/gocore/httpserver.
package httpservermock

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

// Server provides a mock implementation of Server for unit testing.
type Server struct {
	mock.Mock
}

// NewServer creates a new instance of Server.
func NewServer(t mock.TestingT) *Server {
	m := &Server{}
	m.Test(t)
	return m
}

func (m *Server) Engine() *gin.Engine {
	args := m.Called()
	if e, ok := args.Get(0).(*gin.Engine); ok {
		return e
	}
	return nil
}

func (m *Server) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *Server) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
