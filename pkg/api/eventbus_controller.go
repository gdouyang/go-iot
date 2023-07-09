package api

import (
	"errors"
	"fmt"
	"go-iot/pkg/api/realtime"
	device "go-iot/pkg/models/device"
	"net/http"

	logs "go-iot/pkg/logger"

	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/websocket"
)

// websocket实时信息，把监听的数据通过websocket返回
func init() {
	web.Router("/api/eventbus/:productId/:deviceId/:type", &EventbusWebSocketController{}, "get:ProductJoin")
	web.Router("/api/eventbus/:deviceId/:type", &EventbusWebSocketController{}, "get:DeviceJoin")
}

type EventbusWebSocketController struct {
	AuthController
}

func (ctl *EventbusWebSocketController) ProductJoin() {
	productId := ctl.Param(":productId")
	deviceId := ctl.Param(":deviceId")
	typ := ctl.Param(":type")
	ctl.loop(productId, deviceId, typ)
}

func (ctl *EventbusWebSocketController) DeviceJoin() {
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
	dev, err := device.GetDeviceAndCheckCreateId(deviceId, ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.loop(dev.ProductId, deviceId, typ)
}

func (ctl *EventbusWebSocketController) loop(productId string, deviceId string, typ string) {
	if len(productId) == 0 {
		ctl.RespError(errors.New("productId must be present"))
		return
	}
	if len(deviceId) == 0 {
		ctl.RespError(errors.New("deviceId must be present"))
		return
	}
	if len(typ) == 0 {
		ctl.RespError(errors.New("type must be present"))
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
		logs.Errorf("cannot setup WebSocket connection: %v", err)
		ctl.RespError(fmt.Errorf("cannot setup WebSocket connection: %v", err))
		return
	}

	// Join.
	addr := ws.RemoteAddr().String()
	topic := fmt.Sprintf("/device/%s/%s/%s", productId, deviceId, typ)
	realtime.Subscribe(realtime.Subscriber{ProductId: productId, DeviceId: deviceId, Topic: topic, Addr: addr, Conn: ws})
	defer func() {
		realtime.Unsubscribe(realtime.Subscriber{ProductId: productId, DeviceId: deviceId, Topic: topic, Addr: addr})
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
