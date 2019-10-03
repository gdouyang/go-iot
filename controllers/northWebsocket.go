package controllers

import (
	"net/http"

	"go-iot/models"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
)

func init() {
	beego.Router("/north/push", &NorthWebSocketController{}, "post:PushNorth")
	beego.Router("/ws/north", &NorthWebSocketController{}, "get:Join")
}

// 北向websocket 处理北向接口的websocket请求.
type NorthWebSocketController struct {
	beego.Controller
}

// 推送北向WebSocket信息
func (this *NorthWebSocketController) PushNorth() {
	msg := this.GetString("msg")
	publish <- newEvent(models.EVENT_MESSAGE, msg)
}

// 加入方法
func (this *NorthWebSocketController) Join() {
	evt := this.GetString("evt")
	if len(evt) == 0 {
		this.Redirect("/", 302)
		return
	}

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
	addr := Join(models.NORTH, evt, ws)
	defer Leave(addr)

	// Message receive loop.
	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			return
		}
		beego.Info("read message:", string(p))
		// publish <- newEvent(models.EVENT_MESSAGE, addr, string(p))
	}
}
