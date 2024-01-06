package websocketserver

import (
	"encoding/hex"
	"fmt"
	"go-iot/pkg/core"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/gorilla/websocket"
)

const (
	// Connected is ws client status of Connected
	Connected = 1
	// Disconnected is ws client status of Disconnected
	Disconnected = 2
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func newWebsocketSession(conn *websocket.Conn, r *http.Request, wsServer *WebSocketServer, productId string) *WebsocketSession {
	r.ParseForm()
	session := &WebsocketSession{
		id:         fmt.Sprintf("ws%d", time.Now().UnixNano()),
		wsServer:   wsServer,
		conn:       conn,
		header:     r.Header,
		form:       r.Form,
		requestURI: r.RequestURI,
		productId:  productId,
		send:       make(chan *wsMsg, 256),
		statusFlag: Connected,
	}
	return session
}

type wsMsg struct {
	messageType int
	data        []byte
}

type WebsocketSession struct {
	sync.Mutex
	id         string
	wsServer   *WebSocketServer
	conn       *websocket.Conn
	header     http.Header
	form       url.Values
	requestURI string
	deviceId   string
	productId  string
	// Buffered channel of outbound messages.
	send       chan *wsMsg
	statusFlag int32
}

func (s *WebsocketSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *WebsocketSession) GetDeviceId() string {
	return s.deviceId
}

func (s *WebsocketSession) Disconnect() error {
	if s.disconnected() {
		return nil
	}
	core.DelSession(s.deviceId)
	err := s.Close()
	return err
}

func (s *WebsocketSession) Close() error {
	s.Lock()
	if s.disconnected() {
		s.Unlock()
		return nil
	}
	atomic.StoreInt32(&s.statusFlag, Disconnected)
	s.wsServer.removeClient(s.id)
	close(s.send)
	err := s.conn.Close()
	s.Unlock()
	return err
}

func (s *WebsocketSession) SendText(msg string) error {
	s.send <- &wsMsg{messageType: websocket.TextMessage, data: []byte(msg)}
	return nil
}

func (s *WebsocketSession) SendBinary(msg string) error {
	payload, err := hex.DecodeString(msg)
	if err != nil {
		logs.Warnf("Error message, message is not a hex string: %v", err)
		return err
	}
	s.send <- &wsMsg{messageType: websocket.BinaryMessage, data: payload}
	return nil
}

func (c *WebsocketSession) disconnected() bool {
	return atomic.LoadInt32(&c.statusFlag) == Disconnected
}

// readLoop pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (s *WebsocketSession) readLoop() {
	defer func() {
		s.Disconnect()
	}()
	// 处理OnConnect步骤
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
	if s.disconnected() {
		return
	}
	s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logs.Errorf("Error during websocket message reading: %v", err)
			}
			break
		}
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

// writeLoop pumps messages from the hub to the websocket connection.
//
// A goroutine running writeLoop is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *WebsocketSession) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Disconnect()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.WriteMessage(message.messageType, message.data)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
