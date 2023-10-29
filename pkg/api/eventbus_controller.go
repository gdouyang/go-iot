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
	loop := func(ctl *AuthController, productId string, deviceId string, typ string) {
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
	// 监听产品
	web.RegisterAPI("/eventbus/{productId}/{deviceId}/{type}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		productId := ctl.Param("productId")
		deviceId := ctl.Param("deviceId")
		typ := ctl.Param("type")
		loop(ctl, productId, deviceId, typ)
	})
	// 监听设备
	web.RegisterAPI("/eventbus/{deviceId}/{type}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
		loop(ctl, dev.ProductId, deviceId, typ)
	})
}
