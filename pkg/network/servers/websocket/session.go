package websocketsocker

import (
	"encoding/hex"
	"fmt"
	"go-iot/pkg/core"
	"net/http"
	"net/url"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/gorilla/websocket"
)

func newSession(conn *websocket.Conn, r *http.Request, productId string) *WebsocketSession {
	r.ParseForm()
	session := &WebsocketSession{
		id:         fmt.Sprintf("%d", time.Now().UnixNano()),
		conn:       conn,
		header:     r.Header,
		form:       r.Form,
		requestURI: r.RequestURI,
		productId:  productId,
		done:       make(chan struct{}),
	}
	return session
}

type WebsocketSession struct {
	id         string
	conn       *websocket.Conn
	header     http.Header
	form       url.Values
	requestURI string
	deviceId   string
	productId  string
	done       chan struct{}
	isClose    bool
}

func (s *WebsocketSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *WebsocketSession) GetDeviceId() string {
	return s.deviceId
}

func (s *WebsocketSession) Disconnect() error {
	err := s.Close()
	core.DelSession(s.deviceId)
	return err
}

func (s *WebsocketSession) Close() error {
	if s.isClose {
		return nil
	}
	close(s.done)
	s.isClose = true
	err := s.conn.Close()
	core.DelSession(s.deviceId)
	return err
}

func (s *WebsocketSession) SendText(msg string) error {
	err := s.conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		logs.Warnf("Error during websocket SendText: %v", err)
	}
	return err
}

func (s *WebsocketSession) SendBinary(msg string) error {
	payload, err := hex.DecodeString(msg)
	if err != nil {
		logs.Warnf("Error message, message is not a hex string: %v", err)
		return err
	}
	err = s.conn.WriteMessage(websocket.BinaryMessage, payload)
	if err != nil {
		logs.Warnf("Error during websocket SendBinary: %v", err)
	}
	return err
}

func (s *WebsocketSession) readLoop() {
	defer s.Disconnect()
	// The event loop
	sc := core.GetCodec(s.productId)
	sc.OnConnect(&websocketContext{
		BaseContext: core.BaseContext{
			ProductId: s.productId,
			Session:   s,
		},
		header:     s.header,
		form:       s.form,
		requestURI: s.requestURI,
	})
	for {
		select {
		case <-s.done:
			return
		default:
		}
		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			logs.Errorf("Error during websocket message reading: %v", err)
			break
		}
		// logs.Info("Received: %s", message)
		sc := core.GetCodec(s.productId)
		sc.OnMessage(&websocketContext{
			BaseContext: core.BaseContext{
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
