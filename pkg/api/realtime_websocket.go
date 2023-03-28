package api

import (
	"errors"
	"fmt"
	"go-iot/pkg/api/realtime"
	device "go-iot/pkg/models/device"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/websocket"
)

// websocket实时信息，把监听的数据通过websocket返回
func init() {
	web.Router("/api/realtime/:deviceId/:type", &RealtimeWebSocketController{}, "get:Join")

	go realtime.StartLoop()
}

type RealtimeWebSocketController struct {
	AuthController
}

// 加入方法
func (ctl *RealtimeWebSocketController) Join() {
	deviceId := ctl.Param(":deviceId")
	if len(deviceId) == 0 {
		ctl.RespError(errors.New("deviceId must be present"))
		return
	}
	typ := ctl.Param(":type")
	if len(typ) == 0 {
		ctl.RespError(errors.New("type must be present"))
		return
	}
	dev, err := device.GetDeviceMust(deviceId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	// Upgrade from http request to WebSocket.
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	} // use default options
	ws, err := upgrader.Upgrade(ctl.Ctx.ResponseWriter, ctl.Ctx.Request, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(ctl.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		logs.Error("Cannot setup WebSocket connection:", err)
		ctl.RespError(fmt.Errorf("cannot setup WebSocket connection:%v", err))
		return
	}

	// Join.
	addr := ws.RemoteAddr().String()
	topic := fmt.Sprintf("/device/%s/%s/%s", dev.ProductId, dev.Id, typ)
	realtime.Subscribe(realtime.Subscriber{ProductId: dev.ProductId, DeviceId: deviceId, Topic: topic, Addr: addr, Conn: ws})
	defer func() {
		realtime.Unsubscribe(realtime.Subscriber{ProductId: dev.ProductId, DeviceId: deviceId, Topic: topic, Addr: addr})
	}()

	realtime.ListenEventBus(topic)

	// Message receive loop.
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			if web.BConfig.WebConfig.AutoRender {
				ctl.RespOk()
			}
			return
		}
	}
}
