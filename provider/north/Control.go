package north

import (
	"go-iot/models"
	"go-iot/provider/north/sender"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control",
		beego.NSRouter("/:id/switch", &Control{}, "post:Open"),
		beego.NSRouter("/:id/light", &Control{}, "post:Light"),
		beego.NSRouter("/:id/get/online-status", &Control{}, "post:Status"))
	beego.AddNamespace(ns)
}

var (
	northSender sender.NorthSender = sender.NorthSender{CheckAgent: true}
)

type Control struct {
	beego.Controller
}

// 设备开关
func (this *Control) Open() {
	deviceId := this.Ctx.Input.Param(":id")
	data := this.Ctx.Input.RequestBody

	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Data: data, Url: this.Ctx.Input.URL()}
	this.Data["json"] = northSender.Open(request, deviceId)
	this.ServeJSON()
}

// 设备调光
func (this *Control) Light() {
	deviceId := this.Ctx.Input.Param(":id")
	data := this.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Data: data, Url: this.Ctx.Input.URL()}
	this.Data["json"] = northSender.Light(request, deviceId)
	this.ServeJSON()
}

// 状态查询
func (this *Control) Status() {
	deviceId := this.Ctx.Input.Param(":id")
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL()}
	this.Data["json"] = northSender.GetOnlineStatus(request, deviceId)
	this.ServeJSON()
}
