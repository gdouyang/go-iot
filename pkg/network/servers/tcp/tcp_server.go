package tcpserver

import (
	"crypto/tls"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"
	"net"
	"sync"

	logs "go-iot/pkg/logger"
)

func init() {
	servers.RegServer(func() network.NetServer {
		return &TcpServer{
			clients: make(map[string]*TcpSession),
		}
	})
}

var m = map[string]*TcpServer{}

type (
	TcpServer struct {
		sync.RWMutex
		productId string
		spec      *TcpServerSpec
		listener  net.Listener
		tlsCfg    *tls.Config

		// done is the channel for shutdowning this server.
		done    chan struct{}
		clients map[string]*TcpSession
	}
)

func NewServer() *TcpServer {
	return &TcpServer{
		clients: make(map[string]*TcpSession),
	}
}

func (s *TcpServer) Type() network.NetType {
	return network.TCP_SERVER
}

// 开启serverSocket
func (s *TcpServer) Start(network network.NetworkConf) error {

	spec := &TcpServerSpec{}
	err := spec.FromNetwork(network)
	if err != nil {
		return err
	}
	spec.Port = network.Port

	s.productId = network.ProductId
	s.spec = spec
	s.done = make(chan struct{})

	err = s.setListener()
	if err != nil {
		logs.Errorf("tcp server set listener failed: %v", err)
		return err
	}

	go s.run()
	m[network.ProductId] = s
	return nil
}

func (s *TcpServer) setListener() error {
	var l net.Listener
	var err error
	var cfg *tls.Config
	addr := fmt.Sprintf("%s:%d", s.spec.Host, s.spec.Port)
	if s.spec.UseTLS {
		cfg, err = s.spec.TlsConfig()
		if err != nil {
			return fmt.Errorf("invalid tls config for tcp server: %v", err)
		}
		l, err = tls.Listen("tcp", addr, cfg)
		if err != nil {
			return fmt.Errorf("gen tls tcp listener with addr %v and cfg %v failed: %v", addr, cfg, err)
		}
	} else {
		l, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("gen tcp listener with addr %s failed: %v", addr, err)
		}
	}
	s.tlsCfg = cfg
	s.listener = l
	return err
}

func (s *TcpServer) run() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
			}
		} else {
			go s.handleConn(conn)
		}
	}
}

func (s *TcpServer) handleConn(c net.Conn) {
	session := newTcpSession(s, c, s.productId)
	s.Lock()
	s.clients[session.id] = session
	s.Unlock()
	defer session.Disconnect()

	sc := core.GetCodec(s.productId)

	sc.OnConnect(&tcpContext{
		BaseContext: core.BaseContext{
			ProductId: s.productId,
			Session:   session,
		},
	})

	//3.循环读取网络数据流
	go session.readLoop()
	go session.writeLoop()
}

func (s *TcpServer) Reload() error {
	return nil
}

func (s *TcpServer) Stop() error {
	close(s.done)
	s.listener.Close()
	return nil
}

func (b *TcpServer) removeClient(clientID string) {
	b.Lock()
	if val, ok := b.clients[clientID]; ok {
		if val.disconnected() {
			delete(b.clients, clientID)
		}
	}
	b.Unlock()
}

func (s *TcpServer) TotalConnection() int32 {
	l := len(s.clients)
	return int32(l)
}
