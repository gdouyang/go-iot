package websocketsocker

import (
	"encoding/base64"
	"fmt"
	"go-iot/codec"
	"net/http"
	"net/url"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
)

func newSession(conn *websocket.Conn, r *http.Request, productId string) *websocketSession {
	r.ParseForm()
	session := &websocketSession{
		id:         fmt.Sprintf("%d", time.Now().UnixNano()),
		conn:       conn,
		header:     r.Header,
		form:       r.Form,
		requestURI: r.RequestURI,
		productId:  productId,
	}
	return session
}

type websocketSession struct {
	id         string
	conn       *websocket.Conn
	header     http.Header
	form       url.Values
	requestURI string
	deviceId   string
	productId  string
}

func (s *websocketSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *websocketSession) GetDeviceId() string {
	return s.deviceId
}

func (s *websocketSession) Disconnect() error {
	err := s.conn.Close()
	codec.GetSessionManager().DelLocal(s.deviceId)
	return err
}

func (s *websocketSession) SendText(msg string) error {
	err := s.conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logs.Warn("Error during message writing:", err)
	}
	return err
}

func (s *websocketSession) SendBinary(msg string) error {
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
		header:     s.header,
		form:       s.form,
		requestURI: s.requestURI,
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
			header:     s.header,
			form:       s.form,
			requestURI: s.requestURI,
		})
	}
}
