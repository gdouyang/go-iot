package tcpserver

import (
	"go-iot/provider/codec"
	"net"
)

func NewTcpSession(conn net.Conn) codec.Session {
	session := &tcpSession{conn: conn}
	return session
}

type tcpSession struct {
	conn net.Conn
}

func (s *tcpSession) Send(msg interface{}) error {
	s.conn.Write([]byte(msg.(string)))
	return nil
}

func (s *tcpSession) DisConnect() error {
	return s.conn.Close()
}
