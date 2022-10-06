package tcpserver

import (
	"go-iot/codec"
	"net"
)

func newTcpSession(conn net.Conn) codec.Session {
	session := &tcpSession{conn: conn}
	return session
}

type tcpSession struct {
	conn     net.Conn
	deviceId string
}

func (s *tcpSession) Send(msg interface{}) error {
	s.conn.Write([]byte(msg.(string)))
	return nil
}

func (s *tcpSession) Disconnect() error {
	return s.conn.Close()
}

func (s *tcpSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}
