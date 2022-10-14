package tcpserver

import (
	"go-iot/codec"
	"net"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func newTcpSession(s *TcpServerSpec, productId string, conn net.Conn) *tcpSession {
	//2.网络数据流分隔器
	delimeter := newDelimeter(s.Delimeter, conn)
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
}

func (s *tcpSession) Send(msg interface{}) error {
	s.conn.Write([]byte(msg.(string)))
	return nil
}

func (s *tcpSession) Disconnect() error {
	close(s.done)
	return s.conn.Close()
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
				logs.Error("set read timeout failed: %s", s.deviceId)
			}
		}

		//3.1 网络数据流读入 buffer
		data, err := s.delimeter.Read()
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if err != nil {
			logs.Error("read error: " + err.Error())
			break
		}
		sc := codec.GetCodec(s.productId)
		sc.OnMessage(&tcpContext{
			BaseContext: codec.BaseContext{
				DeviceId:  s.GetDeviceId(),
				ProductId: s.productId,
				Session:   s,
			},
			Data: data,
		})
	}
}
