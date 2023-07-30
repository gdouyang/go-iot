package tcpclient

import (
	"encoding/hex"
	"go-iot/pkg/core"
	tcpserver "go-iot/pkg/network/servers/tcp"
	"net"
	"strings"
	"time"

	logs "go-iot/pkg/logger"
)

func newTcpSession(deviceId string, s *TcpClientSpec, productId string, conn net.Conn) *TcpSession {
	//2.网络数据流分隔器
	delimeter := tcpserver.NewDelimeter(s.Delimeter, conn)
	session := &TcpSession{
		deviceId:  deviceId,
		productId: productId,
		conn:      conn, delimeter: delimeter,
		done: make(chan struct{}),
	}
	session.deviceOnline(deviceId)
	return session
}

type TcpSession struct {
	conn      net.Conn
	deviceId  string
	productId string
	keepalive uint16
	delimeter tcpserver.Delimeter
	done      chan struct{}
	isClose   bool
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

func (s *TcpSession) Disconnect() error {
	if s.isClose {
		return nil
	}
	close(s.done)
	s.isClose = true
	err := s.conn.Close()
	core.DelSession(s.deviceId)
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

func (s *TcpSession) deviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		core.PutSession(deviceId, s)
	}
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
