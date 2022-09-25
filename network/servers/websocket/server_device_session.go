package websocketsocker

import (
	"go-iot/codec"

	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
)

func newSession(conn *websocket.Conn) codec.Session {
	session := &websocketSession{conn: conn}
	return session
}

type websocketSession struct {
	conn     *websocket.Conn
	deviceId string
}

func (s *websocketSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
	codec.GetSessionManager().PutSession(deviceId, s)
}

func (s *websocketSession) Send(msg interface{}) error {
	err := s.conn.WriteMessage(websocket.BinaryMessage, msg.([]byte))
	if err != nil {
		logs.Warn("Error during message writing:", err)
	}
	return err
}

func (s *websocketSession) DisConnect() error {
	return s.conn.Close()
}
