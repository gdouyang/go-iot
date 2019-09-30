package routers

import (
	"go-iot/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	// WebSocket.
	beego.Router("/ws/join", &controllers.WebSocketController{}, "get:Join")
}
