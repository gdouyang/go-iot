package httpserver

import (
	"crypto/tls"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/eventbus"
	"go-iot/pkg/network/servers"
	"net"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	servers.RegServer(func() core.NetServer {
		return NewServer()
	})
}

type (
	HttpServer struct {
		productId   string
		spec        *HttpServerSpec
		server      *http.Server
		pathmatcher eventbus.AntPathMatcher
	}
)

func NewServer() *HttpServer {
	return &HttpServer{
		pathmatcher: *eventbus.NewAntPathMatcher(),
	}
}

func (s *HttpServer) Type() core.NetType {
	return core.HTTP_SERVER
}

func (s *HttpServer) Start(network core.NetworkConf) error {
	spec := &HttpServerSpec{}
	err := spec.FromNetwork(network)
	if err != nil {
		return err
	}
	spec.Port = network.Port

	if len(spec.Routers) == 0 {
		spec.Routers = append(spec.Routers, Router{Url: "/**"})
	}

	s.productId = network.ProductId
	s.spec = spec
	addr := fmt.Sprintf("%s:%d", spec.Host, spec.Port)

	s.server = &http.Server{
		Addr:    addr,
		Handler: s,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	var tlsConfig *tls.Config
	if spec.UseTLS {
		tlsConfig, err = spec.TlsConfig()
		if err != nil {
			return err
		}
	}

	go func() {
		var err error
		if spec.UseTLS {
			s.server.TLSConfig = tlsConfig
			err = s.server.ServeTLS(listener, "", "")
		} else {
			err = s.server.Serve(listener)
		}
		if err != nil {
			logs.Error(err)
		}
	}()
	return nil
}

func (s *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	allow := false
	for _, route := range s.spec.Routers {
		if s.pathmatcher.Match(route.Url, r.RequestURI) {
			allow = true
			break
		}
	}
	if !allow {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	r.ParseForm()
	session := newSession(w, r, s.productId)

	session.readData()
}

func (s *HttpServer) Reload() error {
	return nil
}

func (s *HttpServer) Stop() error {
	return s.server.Close()
}

func (s *HttpServer) TotalConnection() int32 {
	return 0
}
