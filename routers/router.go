package routers

import (
	"go-iot/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	// WebSocket.
	beego.Router("/ws/echo", &controllers.EchoWebSocketController{}, "get:Join")
	beego.Router("/ws/north", &controllers.NorthWebSocketController{}, "get:Join")
	beego.Router("/north/push", &controllers.NorthWebSocketController{}, "post:PushNorth")
}
