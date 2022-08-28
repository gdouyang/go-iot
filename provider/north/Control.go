package north

import (
	"go-iot/models"
	"go-iot/provider/north/sender"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	ns := web.NewNamespace("/north/control",
		web.NSRouter("/:id/switch", &Control{}, "post:Open"),
		web.NSRouter("/:id/light", &Control{}, "post:Light"),
		web.NSRouter("/:id/get/online-status", &Control{}, "post:Status"))
	web.AddNamespace(ns)
}

var (
	northSender sender.NorthSender = sender.NorthSender{CheckAgent: true}
)

type Control struct {
	web.Controller
}

// 设备开关
func (ctl *Control) Open() {
	deviceId := ctl.Ctx.Input.Param(":id")
	data := ctl.Ctx.Input.RequestBody

	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Data: data, Url: ctl.Ctx.Input.URL()}
	ctl.Data["json"] = northSender.Open(request, deviceId)
	ctl.ServeJSON()
}

// 设备调光
func (ctl *Control) Light() {
	deviceId := ctl.Ctx.Input.Param(":id")
	data := ctl.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Data: data, Url: ctl.Ctx.Input.URL()}
	ctl.Data["json"] = northSender.Light(request, deviceId)
	ctl.ServeJSON()
}

// 状态查询
func (ctl *Control) Status() {
	deviceId := ctl.Ctx.Input.Param(":id")
	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Url: ctl.Ctx.Input.URL()}
	ctl.Data["json"] = northSender.GetOnlineStatus(request, deviceId)
	ctl.ServeJSON()
}
