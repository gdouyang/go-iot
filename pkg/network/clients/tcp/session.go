package tcpclient

import (
	"encoding/hex"
	"go-iot/pkg/core"
	tcpserver "go-iot/pkg/network/servers/tcp"
	"net"
	"strings"
	"sync"
	"time"

	logs "go-iot/pkg/logger"
)

func newTcpSession(deviceId string, s *TcpClientSpec, productId string, conn net.Conn) *TcpSession {
	deviceId = strings.TrimSpace(deviceId)
	//2.网络数据流分隔器
	delimeter := tcpserver.NewDelimeter(s.Delimeter, conn)
	session := &TcpSession{
		deviceId:  deviceId,
		productId: productId,
		conn:      conn,
		delimeter: delimeter,
		done:      make(chan struct{}),
		info:      map[string]any{},
	}
	if len(deviceId) > 0 {
		core.PutSession(deviceId, session, true)
	}
	return session
}

type TcpSession struct {
	mu        sync.Mutex
	conn      net.Conn
	deviceId  string
	productId string
	keepalive uint16
	delimeter tcpserver.Delimeter
	done      chan struct{}
	isClose   bool
	info      map[string]any
}

func (s *TcpSession) Send(msg string) error {
	_, err := s.conn.Write([]byte(msg))
	if err != nil {
		logs.Errorf("tcpclient Send error: %v", err)
	}
	return err
}

func (s *TcpSession) SendHex(msgHex string) error {
	b, err := hex.DecodeString(msgHex)
	if err != nil {
		logs.Errorf("tcpclient hex decode error: %v", err)
		return err
	}
	_, err = s.conn.Write(b)
	if err != nil {
		logs.Errorf("tcpclient SendHex error: %v", err)
	}
	return err
}

// Disconnect 幂等：API Close 与 readLoop defer Disconnect 可能并发触发，
// 无锁读/写 isClose 会发生 double close(s.done) panic
func (s *TcpSession) Disconnect() error {
	s.mu.Lock()
	if s.isClose {
		s.mu.Unlock()
		return nil
	}
	s.isClose = true
	close(s.done)
	s.mu.Unlock()
	err := s.conn.Close()
	core.DelSessionByUserDisconnect(s.deviceId)
	return err
}

func (s *TcpSession) Close() error {
	return s.Disconnect()
}

func (s *TcpSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *TcpSession) GetDeviceId() string {
	return s.deviceId
}

func (s *TcpSession) GetConInfo() map[string]any {
	// GetConInfo 写共享 s.info：并发 API 查询会写该 map，不持锁会
	// fatal: concurrent map read and map write
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info["localAddr"] = func() string {
		if s.conn != nil {
			return s.conn.LocalAddr().String()
		}
		return "unknown" // 或者返回其他适当的默认值
	}()
	s.info["remoteAddr"] = func() string {
		if s.conn != nil {
			return s.conn.RemoteAddr().String()
		}
		return "unknown" // 或者返回其他适当的默认值
	}()
	return s.info
}

func (s *TcpSession) readLoop() {
	keepAlive := time.Duration(s.keepalive) * time.Second
	timeOut := keepAlive + keepAlive/2
	for {
		select {
		case <-s.done:
			return
		default:
		}

		if keepAlive > 0 {
			if err := s.conn.SetDeadline(time.Now().Add(timeOut)); err != nil {
				logs.Errorf("tcpclient set read timeout failed: %s", s.deviceId)
			}
		}

		//3.1 网络数据流读入 buffer
		data, err := s.delimeter.Read()
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if err != nil {
			logs.Debugf("tcpclient read error: %v", err)
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
