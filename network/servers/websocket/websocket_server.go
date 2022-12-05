package websocketsocker

import (
	"crypto/tls"
	"fmt"
	"go-iot/codec"
	"go-iot/codec/eventbus"
	"go-iot/network/servers"
	"net"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
)

func init() {
	servers.RegServer(func() codec.NetServer {
		return NewServer()
	})
}

var upgrader = websocket.Upgrader{} // use default options

type (
	WebSocketServer struct {
		productId   string
		spec        *WebsocketServerSpec
		server      *http.Server
		pathmatcher eventbus.AntPathMatcher
	}
)

func NewServer() *WebSocketServer {
	return &WebSocketServer{
		pathmatcher: *eventbus.NewAntPathMatcher(),
	}
}

func (s *WebSocketServer) Type() codec.NetServerType {
	return codec.WEBSOCKET_SERVER
}

func (s *WebSocketServer) Start(network codec.NetworkConf) error {
	spec := &WebsocketServerSpec{}
	err := spec.FromJson(network.Configuration)
	if err != nil {
		return err
	}
	spec.Port = network.Port

	if len(spec.Routers) == 0 {
		spec.Routers = append(spec.Routers, Router{Id: 1, Url: "/**"})
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

func (s *WebSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	// Upgrade our raw HTTP connection to a websocket based one
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logs.Error("Error during websocket connection upgradation:", err)
		return
	}

	session := newSession(conn, r, s.productId)
	session.readLoop()
}

func (s *WebSocketServer) Reload() error {
	return nil
}

func (s *WebSocketServer) Stop() error {
	return nil
}

func (s *WebSocketServer) TotalConnection() int32 {
	return 0
}
