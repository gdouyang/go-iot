package tcpclient

import (
	"encoding/hex"
	"go-iot/pkg/core"
	tcpserver "go-iot/pkg/network/servers/tcp"
	"net"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func newTcpSession(deviceId string, s *TcpClientSpec, productId string, conn net.Conn) *tcpSession {
	//2.网络数据流分隔器
	delimeter := tcpserver.NewDelimeter(s.Delimeter, conn)
	session := &tcpSession{
		deviceId:  deviceId,
		productId: productId,
		conn:      conn, delimeter: delimeter,
		done: make(chan struct{}),
	}
	session.deviceOnline(deviceId)
	return session
}

type tcpSession struct {
	conn      net.Conn
	deviceId  string
	productId string
	keepalive uint16
	delimeter tcpserver.Delimeter
	done      chan struct{}
	isClose   bool
}

func (s *tcpSession) Send(msg string) error {
	_, err := s.conn.Write([]byte(msg))
	if err != nil {
		logs.Error("tcpclient Send error:", err)
	}
	return err
}

func (s *tcpSession) SendHex(msgHex string) error {
	b, err := hex.DecodeString(msgHex)
	if err != nil {
		logs.Error("tcpclient hex decode error:", err)
		return err
	}
	_, err = s.conn.Write(b)
	if err != nil {
		logs.Error("tcpclient SendHex error:", err)
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

func (s *tcpSession) deviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		core.PutSession(deviceId, s)
	}
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
				logs.Error("tcpclient set read timeout failed: %s", s.deviceId)
			}
		}

		//3.1 网络数据流读入 buffer
		data, err := s.delimeter.Read()
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if err != nil {
			logs.Debug("tcpclient read error: " + err.Error())
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
