package httpserver

import (
	"fmt"
	"go-iot/codec"
	"net"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

type (
	HttpServer struct {
		productId string
		spec      *HttpServerSpec
		server    *http.Server
	}
)

func ServerStart(network codec.Network) {
	spec := &HttpServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port

	if len(spec.Paths) == 0 {
		spec.Paths = append(spec.Paths, "/")
	}

	s := &HttpServer{
		productId: network.ProductId,
		spec:      spec,
	}
	addr := fmt.Sprintf("%s:%d", spec.Host, spec.Port)

	s.server = &http.Server{
		Addr:    addr,
		Handler: s,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logs.Error(err)
		return
	}

	codec.NewCodec(network)

	go func() {
		var err error
		if spec.UseTLS {
			tlsConfig, _ := spec.TlsConfig()
			s.server.TLSConfig = tlsConfig
			err = s.server.ServeTLS(listener, "", "")
		} else {
			err = s.server.Serve(listener)
		}
		if err != nil {
			logs.Error(err)
		}
	}()
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
