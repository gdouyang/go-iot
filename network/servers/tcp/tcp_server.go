package tcpserver

import (
	"crypto/tls"
	"fmt"
	"go-iot/codec"
	"net"

	"github.com/beego/beego/v2/core/logs"
)

var m = map[string]*TcpServer{}

type (
	TcpServer struct {
		productId string
		spec      *TcpServerSpec
		listener  net.Listener
		tlsCfg    *tls.Config

		// done is the channel for shutdowning this server.
		done chan struct{}
	}
)

// 开启serverSocket
func ServerSocket(network codec.Network) bool {

	spec := &TcpServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port

	server := &TcpServer{
		productId: network.ProductId,
		spec:      spec,
		done:      make(chan struct{}),
	}
	err := server.setListener()
	if err != nil {
		logs.Error("mqtt broker set listener failed: %v", err)
		return false
	}

	// create codec
	codec.NewCodec(network)

	go server.run()
	m[network.ProductId] = server
	return true
}

func (s *TcpServer) setListener() error {
	var l net.Listener
	var err error
	var cfg *tls.Config
	addr := fmt.Sprintf("%s:%d", s.spec.Host, s.spec.Port)
	if s.spec.UseTLS {
		cfg, err = s.spec.TlsConfig()
		if err != nil {
			return fmt.Errorf("invalid tls config for mqtt proxy: %v", err)
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
	session := newTcpSession(s.spec, s.productId, c)
	defer session.Disconnect()

	sc := codec.GetCodec(s.productId)

	sc.OnConnect(&tcpContext{
		BaseContext: codec.BaseContext{
			ProductId: s.productId,
			Session:   session,
		},
	})

	//3.循环读取网络数据流
	session.readLoop()
}

func (b *TcpServer) Close() {
	close(b.done)
	b.listener.Close()
}
