package api

import (
	"fmt"
	"go-iot/pkg/api/realtime"
	"go-iot/pkg/api/web"
	device "go-iot/pkg/models/device"
	"net/http"

	logs "go-iot/pkg/logger"

	"github.com/gorilla/websocket"
)

// websocket实时信息，把监听的数据通过websocket返回
func init() {
	web.RegisterAPI("/eventbus/{productId}/{deviceId}/{type}", "GET", &EventbusWebSocketController{}, "ProductJoin")
	web.RegisterAPI("/eventbus/{deviceId}/{type}", "GET", &EventbusWebSocketController{}, "DeviceJoin")
}

type EventbusWebSocketController struct {
	AuthController
}

func (ctl *EventbusWebSocketController) ProductJoin() {
	productId := ctl.Param("productId")
	deviceId := ctl.Param("deviceId")
	typ := ctl.Param("type")
	ctl.loop(productId, deviceId, typ)
}

func (ctl *EventbusWebSocketController) DeviceJoin() {
	deviceId := ctl.Param("deviceId")
	if len(deviceId) == 0 {
		ctl.RespErrorParam("deviceId")
		return
	}
	typ := ctl.Param("type")
	dev, err := device.GetDeviceAndCheckCreateId(deviceId, ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.loop(dev.ProductId, deviceId, typ)
}

func (ctl *EventbusWebSocketController) loop(productId string, deviceId string, typ string) {
	if len(productId) == 0 {
		ctl.RespErrorParam("productId")
		return
	}
	if len(deviceId) == 0 {
		ctl.RespErrorParam("deviceId")
		return
	}
	if len(typ) == 0 {
		ctl.RespErrorParam("type")
		return
	}
	// Upgrade from http request to WebSocket.
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	} // use default options
	ws, err := upgrader.Upgrade(ctl.ResponseWriter, ctl.Request, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(ctl.ResponseWriter, "Not a websocket handshake", 400)
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
	go func() {
		defer func() {
			realtime.Unsubscribe(realtime.Subscriber{ProductId: productId, DeviceId: deviceId, Topic: topic, Addr: addr})
		}()

		// Message receive loop.
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
}
