package xixun

import (
	"go-iot/models"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control/xixun/v1",
		beego.NSRouter("/:id/screenShot", &XiXunLedController{}, "post:ScreenShot"))
	beego.AddNamespace(ns)
}

type XiXunLedController struct {
	beego.Controller
}

// 设备开关
func (this *XiXunLedController) ScreenShot() {
	deviceId := this.Ctx.Input.Param(":id")
	beego.Info("deviceId=", deviceId)
	device, err := models.GetDevice(deviceId)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		operResp := ProviderImplXiXunLed.ScreenShot(device.Sn)
		this.Data["json"] = models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
	}

	this.ServeJSON()
}
