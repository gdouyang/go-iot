package web

import (
	"go-iot/pkg/logger"
	"net/http"
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

// MustNewServer creates an api server.
func MustNewServer(addr string) *Server {
	s := &Server{}
	s.router = newDynamicMux(s)
	s.server = http.Server{Addr: addr, Handler: s.router}

	logger.Infof("api server running in %s", addr)
	s.server.ListenAndServe()

	return s
}
