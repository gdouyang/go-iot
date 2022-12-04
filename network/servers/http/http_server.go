package httpserver

import (
	"crypto/tls"
	"fmt"
	"go-iot/codec"
	"go-iot/network/servers"
	"net"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	servers.RegServer(func() codec.NetServer {
		return &HttpServer{}
	})
}

type (
	HttpServer struct {
		productId string
		spec      *HttpServerSpec
		server    *http.Server
	}
)

func NewServer() *HttpServer {
	return &HttpServer{}
}

func (s *HttpServer) Type() codec.NetServerType {
	return codec.HTTP_SERVER
}

func (s *HttpServer) Start(network codec.NetworkConf) error {
	spec := &HttpServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port

	if len(spec.Paths) == 0 {
		spec.Paths = append(spec.Paths, "/")
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

	codec.NewCodec(network)

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
	for _, path := range s.spec.Paths {
		if strings.HasPrefix(r.RequestURI, path) {
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
	return nil
}

func (s *HttpServer) TotalConnection() int32 {
	return 0
}
