package routers

import (
	_ "go-iot/controllers"
	// "github.com/astaxie/beego"
)

func init() {

	// WebSocket.
	// beego.Router("/ws/echo", &controllers.EchoWebSocketController{}, "get:Join")
	// beego.Router("/ws/north", &controllers.NorthWebSocketController{}, "get:Join")
	// beego.Router("/north/push", &controllers.NorthWebSocketController{}, "post:PushNorth")
}
