package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go-iot/pkg/logger"
)

type (
	// Server is the api server.
	Server struct {
		server http.Server
		router *dynamicMux
	}

	// Entry is the entry of API.
	Entry struct {
		Path          string
		Method        string
		Controller    ControllerInterface
		HandlerMethod string
		Handler       http.HandlerFunc
	}
)

// NewServer creates an api server without starting it.
func NewServer(addr string) *Server {
	s := &Server{}
	s.router = newDynamicMux(s)
	s.server = http.Server{Addr: addr, Handler: s.router}
	return s
}

// Start begins ListenAndServe in a background goroutine (non-blocking).
func (s *Server) Start() error {
	logger.Infof("api server running in %s", s.server.Addr)
	go func() {
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Errorf("api server error: %v", err)
		}
	}()
	return nil
}

// Shutdown gracefully stops the API server.
func (s *Server) Shutdown(ctx context.Context) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}
	logger.Infof("api server shutting down")
	return s.server.Shutdown(ctx)
}

// MustNewServer creates and starts an api server, then blocks until the server exits.
// Deprecated for app assembly: prefer NewServer + Start + signal-driven Shutdown.
func MustNewServer(addr string) *Server {
	s := NewServer(addr)
	logger.Infof("api server running in %s", addr)
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Errorf("api server error: %v", err))
	}
	return s
}
