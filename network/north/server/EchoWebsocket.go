package server

import (
	"net/http"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/websocket"
)

func init() {
	web.Router("/ws/echo", &EchoWebSocketController{}, "get:Join")
}

// EchoWebSocketController处理浏览器的Echo WebSocket请求.
type EchoWebSocketController struct {
	web.Controller
}

// 加入方法
func (this *EchoWebSocketController) Join() {

	// Upgrade from http request to WebSocket.
	ws, err := websocket.Upgrade(this.Ctx.ResponseWriter, this.Ctx.Request, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(this.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		logs.Error("Cannot setup WebSocket connection:", err)
		return
	}

	// Join.
	addr := Join(ECHO, "", ws)
	defer Leave(addr)

	// Message receive loop.
	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			return
		}
		logs.Info("read message:", string(p))
		// publish <- newEvent(models.EVENT_MESSAGE, addr, string(p), models.TARGET_ECHO)
	}
}
