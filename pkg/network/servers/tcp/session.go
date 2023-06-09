package tcpserver

import (
	"encoding/hex"
	"go-iot/pkg/core"
	"net"
	"time"

	logs "go-iot/pkg/logger"
)

func newTcpSession(s *TcpServerSpec, productId string, conn net.Conn) *tcpSession {
	//2.网络数据流分隔器
	delimeter := NewDelimeter(s.Delimeter, conn)
	session := &tcpSession{
		productId: productId,
		conn:      conn, delimeter: delimeter,
		done: make(chan struct{}),
	}
	return session
}

type tcpSession struct {
	conn      net.Conn
	deviceId  string
	productId string
	keepalive uint16
	delimeter Delimeter
	done      chan struct{}
	isClose   bool
}

func (s *tcpSession) Send(msg string) error {
	_, err := s.conn.Write([]byte(msg))
	if err != nil {
		logs.Errorf("tcp Send error: %v", err)
	}
	return err
}

func (s *tcpSession) SendHex(msgHex string) error {
	b, err := hex.DecodeString(msgHex)
	if err != nil {
		logs.Errorf("tcp hex decode error: %v", err)
		return err
	}
	_, err = s.conn.Write(b)
	if err != nil {
		logs.Errorf("tcp SendHex error: %v", err)
	}
	return err
}

func (s *tcpSession) Disconnect() error {
	if s.isClose {
		return nil
	}
	close(s.done)
	s.isClose = true
	err := s.conn.Close()
	core.DelSession(s.deviceId)
	return err
}

func (s *tcpSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *tcpSession) GetDeviceId() string {
	return s.deviceId
}

func (s *tcpSession) readLoop() {
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
