package coapserver

import (
	"bytes"
	"fmt"
	"go-iot/pkg/eventbus"
	logs "go-iot/pkg/logger"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/udp"
	udpServer "github.com/plgd-dev/go-coap/v3/udp/server"
)

func init() {
	servers.RegServer(func() network.NetServer {
		return NewServer()
	})
}

type (
	CoapServer struct {
		productId   string
		spec        *CoapServerSpec
		server      *udpServer.Server
		pathmatcher eventbus.AntPathMatcher
	}
)

func NewServer() *CoapServer {
	return &CoapServer{
		pathmatcher: *eventbus.NewAntPathMatcher(),
	}
}

func (s *CoapServer) Type() network.NetType {
	return network.COAP_SERVER
}

func (s *CoapServer) Start(network network.NetworkConf) error {
	spec := &CoapServerSpec{}
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
	l, err := net.NewListenUDP("udp", addr)
	if err != nil {
		return err
	}
	go func() {
		server := udp.NewServer(options.WithMux(s))
		err := server.Serve(l)
		if err != nil {
			logs.Errorf("start coap server error: %v", err)
		}
		s.server = server
	}()
	logs.Infof("coap server start: %s %s", s.productId, addr)
	return nil
}

func (s *CoapServer) ServeCOAP(w mux.ResponseWriter, r *mux.Message) {
	allow := false
	path, err := r.Path()
	if err != nil {
		logs.Errorf("cannot set response: %v", err)
	}
	for _, route := range s.spec.Routers {
		if s.pathmatcher.Match(route.Url, path) {
			allow = true
			break
		}
	}
	if !allow {
		err := sendResponse(w, codes.NotFound, message.TextPlain, "")
		if err != nil {
			logs.Errorf("cannot set response: %v", err)
		}
		return
	}

	session := newSession(w, r, s.productId)

	session.readData()
}

func (s *CoapServer) Reload() error {
	return nil
}

func (s *CoapServer) Stop() error {
	s.server.Stop()
	return nil
}

func (s *CoapServer) TotalConnection() int32 {
	return 0
}

func sendResponse(w mux.ResponseWriter, code codes.Code, contentFormat message.MediaType, msg string) error {
	err := w.SetResponse(code, contentFormat, bytes.NewReader([]byte(msg)))
	return err
}
