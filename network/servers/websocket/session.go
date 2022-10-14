package websocketsocker

import (
	"encoding/base64"
	"encoding/json"
	"go-iot/codec"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
)

func newSession(conn *websocket.Conn, r *http.Request, productId string) *websocketSession {
	r.ParseForm()
	session := &websocketSession{
		conn:      conn,
		r:         r,
		productId: productId,
	}
	return session
}

type websocketSession struct {
	conn      *websocket.Conn
	r         *http.Request
	deviceId  string
	productId string
}

func (s *websocketSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *websocketSession) GetDeviceId() string {
	return s.deviceId
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
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, payload)
	if err != nil {
		logs.Warn("Error during message writing:", err)
	}
	return err
}

func (s *websocketSession) readLoop() {
	defer s.Disconnect()
	// The event loop
	sc := codec.GetCodec(s.productId)
	sc.OnConnect(&websocketContext{
		BaseContext: codec.BaseContext{
			ProductId: s.productId,
			Session:   s,
		},
		header:     s.r.Header,
		form:       s.r.Form,
		requestURI: s.r.RequestURI,
	})
	for {
		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			logs.Error("Error during message reading:", err)
			break
		}
		// logs.Info("Received: %s", message)
		sc := codec.GetCodec(s.productId)
		sc.OnMessage(&websocketContext{
			BaseContext: codec.BaseContext{
				DeviceId:  s.GetDeviceId(),
				ProductId: s.productId,
				Session:   s,
			},
			Data:       message,
			msgType:    messageType,
			header:     s.r.Header,
			form:       s.r.Form,
			requestURI: s.r.RequestURI,
		})
	}
}
