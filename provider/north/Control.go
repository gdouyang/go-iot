package north

import (
	"encoding/json"
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
	byteReq := this.Ctx.Input.RequestBody
	this.Data["json"] = northSender.Open(byteReq, deviceId)
	this.ServeJSON()
}

// 设备调光
func (this *Control) Light() {
	deviceId := this.Ctx.Input.Param(":id")
	byteReq := this.Ctx.Input.RequestBody

	this.Data["json"] = northSender.Light(byteReq, deviceId)
	this.ServeJSON()
}

// 状态查询
func (this *Control) Status() {
	var ob map[string]interface{}
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	this.Data["json"] = &ob
	this.ServeJSON()
}
