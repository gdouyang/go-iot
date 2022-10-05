package websocketsocker

import (
	"encoding/base64"
	"encoding/json"
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
	codec.GetSessionManager().Put(deviceId, s)
}

func (s *websocketSession) Send(msg interface{}) error {
	var err error
	switch t := msg.(type) {
	case string:
		err = s.conn.WriteMessage(websocket.TextMessage, []byte(t))
	case map[string]interface{}:
		b, err1 := json.Marshal(t)
		if err1 != nil {
			logs.Warn("map to json string error:", err)
		}
		err = s.conn.WriteMessage(websocket.TextMessage, b)
	default:
		logs.Warn("unsupport msg:", msg)
	}
	if err != nil {
		logs.Warn("Error during message writing:", err)
	}
	return err
}

func (s *websocketSession) Disconnect() error {
	return s.conn.Close()
}

func (s *websocketSession) SendBinaryMessage(msg string) error {
	payload, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		logs.Warn("Error message, message is not a base64 string:", err)
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, payload)
	if err != nil {
		logs.Warn("Error during message writing:", err)
	}
	return err
}
