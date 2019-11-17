package server

import (
	"net/http"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
)

func init() {
	beego.Router("/ws/echo", &EchoWebSocketController{}, "get:Join")
}

// EchoWebSocketController处理浏览器的Echo WebSocket请求.
type EchoWebSocketController struct {
	beego.Controller
}

// 加入方法
func (this *EchoWebSocketController) Join() {

	// Upgrade from http request to WebSocket.
	ws, err := websocket.Upgrade(this.Ctx.ResponseWriter, this.Ctx.Request, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(this.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		beego.Error("Cannot setup WebSocket connection:", err)
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
		beego.Info("read message:", string(p))
		// publish <- newEvent(models.EVENT_MESSAGE, addr, string(p), models.TARGET_ECHO)
	}
}
