package tcpserver

import (
	"encoding/hex"
	"fmt"
	"go-iot/pkg/core"
	"net"
	"sync"
	"sync/atomic"
	"time"

	logs "go-iot/pkg/logger"
)

const (
	// Connected is ws client status of Connected
	Connected = 1
	// Disconnected is ws client status of Disconnected
	Disconnected = 2
)

func newTcpSession(server *TcpServer, conn net.Conn, productId string) *TcpSession {
	//2.网络数据流分隔器
	delimeter := NewDelimeter(server.spec.Delimeter, conn)
	session := &TcpSession{
		id:         fmt.Sprintf("tcp%d", time.Now().UnixNano()),
		tcpServer:  server,
		conn:       conn,
		productId:  productId,
		delimeter:  delimeter,
		send:       make(chan []byte, 256),
		statusFlag: Connected,
	}
	return session
}

type TcpSession struct {
	sync.Mutex
	id        string
	tcpServer *TcpServer
	conn      net.Conn
	productId string
	deviceId  string
	keepalive uint16
	delimeter Delimeter
	// Buffered channel of outbound messages.
	send       chan []byte
	statusFlag int32
}

func (s *TcpSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *TcpSession) GetDeviceId() string {
	return s.deviceId
}
func (s *TcpSession) GetInfo() map[string]any {
	return map[string]any{
		"localAddr": func() string {
			if s.conn != nil {
				return s.conn.LocalAddr().String()
			}
			return "unknown" // 或者返回其他适当的默认值
		}(),
		"remoteAddr": func() string {
			if s.conn != nil {
				return s.conn.RemoteAddr().String()
			}
			return "unknown" // 或者返回其他适当的默认值
		}(),
	}
}

func (s *TcpSession) Disconnect() error {
	if s.disconnected() {
		return nil
	}
	core.DelSession(s.deviceId)
	err := s.Close()
	return err
}

func (s *TcpSession) Close() error {
	s.Lock()
	if s.disconnected() {
		s.Unlock()
		return nil
	}
	atomic.StoreInt32(&s.statusFlag, Disconnected)
	s.tcpServer.removeClient(s.id)
	close(s.send)
	err := s.conn.Close()
	s.Unlock()
	return err
}

func (s *TcpSession) Send(msg string) error {
	s.send <- []byte(msg)
	return nil
}

func (s *TcpSession) SendHex(msgHex string) error {
	b, err := hex.DecodeString(msgHex)
	if err != nil {
		logs.Errorf("tcp hex decode error: %v", err)
		return err
	}
	s.send <- b
	return nil
}

func (c *TcpSession) disconnected() bool {
	return atomic.LoadInt32(&c.statusFlag) == Disconnected
}

func (s *TcpSession) readLoop() {
	defer s.Disconnect()

	// 处理OnConnect步骤
	sc := core.GetCodec(s.productId)
	sc.OnConnect(&tcpContext{
		BaseContext: core.BaseContext{
			ProductId: s.productId,
			Session:   s,
		},
	})
	if s.disconnected() {
		return
	}

	keepAlive := time.Duration(s.keepalive) * time.Second
	timeOut := keepAlive + keepAlive/2
	for {
		if keepAlive > 0 {
			if err := s.conn.SetDeadline(time.Now().Add(timeOut)); err != nil {
				logs.Errorf("set read timeout failed: %s", s.deviceId)
			}
		}

		//3.1 网络数据流读入 buffer
		data, err := s.delimeter.Read()
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if err != nil {
			logs.Debugf("tcp server read error: %v", err)
			break
		}
		sc := core.GetCodec(s.productId)
		sc.OnMessage(&tcpContext{
			BaseContext: core.BaseContext{
				DeviceId:  s.GetDeviceId(),
				ProductId: s.productId,
				Session:   s,
			},
			Data: data,
		})
	}
}

func (c *TcpSession) writeLoop() {
	defer func() {
		c.Disconnect()
	}()
	for {
		message, ok := <-c.send
		if !ok {
			// The hub closed the channel.
			return
		}
		_, err := c.conn.Write(message)
		if err != nil {
			logs.Errorf("tcp write error: %v", err)
		}
	}
}
