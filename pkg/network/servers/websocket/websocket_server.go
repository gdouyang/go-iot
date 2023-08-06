package websocketsocker

import (
	"crypto/tls"
	"fmt"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"
	"net"
	"net/http"

	logs "go-iot/pkg/logger"

	"github.com/gorilla/websocket"
)

func init() {
	servers.RegServer(func() network.NetServer {
		return NewServer()
	})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
} // use default options

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

func (s *WebSocketServer) Type() network.NetType {
	return network.WEBSOCKET_SERVER
}

func (s *WebSocketServer) Start(network network.NetworkConf) error {
	spec := &WebsocketServerSpec{}
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
			logs.Errorf(err.Error())
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
		logs.Errorf("Error during [%s] websocket connection upgradation: %v", s.productId, err)
		return
	}

	session := newSession(conn, r, s.productId)
	go session.readLoop()
	go session.writeLoop()
}

func (s *WebSocketServer) Reload() error {
	return nil
}

func (s *WebSocketServer) Stop() error {
	return s.server.Close()
}

func (s *WebSocketServer) TotalConnection() int32 {
	return 0
}
