package xixuncontroller

import (
	"go-iot/models"
	"go-iot/provider/xixun/sender"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control/xixun/v1",
		beego.NSRouter("/:id/screenShot", &XiXunLedController{}, "post:ScreenShot"),
		beego.NSRouter("/:id/fileUpload", &XiXunLedController{}, "post:FileUpload"),
		beego.NSRouter("/:id/ledPlay", &XiXunLedController{}, "post:LedPlay"),
		beego.NSRouter("/:id/msgPublish", &XiXunLedController{}, "post:MsgPublish"),
		beego.NSRouter("/:id/msgClear", &XiXunLedController{}, "post:Clear"))
	beego.AddNamespace(ns)
}

type XiXunLedController struct {
	beego.Controller
}

// LED截图
func (this *XiXunLedController) ScreenShot() {
	deviceId := this.Ctx.Input.Param(":id")

	xSender := sender.XixunSender{CheckAgent: true}
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL()}
	this.Data["json"] = xSender.ScreenShot(request, deviceId)

	this.ServeJSON()
}

// LED播放文件上传
func (this *XiXunLedController) FileUpload() {
	deviceId := this.Ctx.Input.Param(":id")
	data := this.Ctx.Input.RequestBody

	xSender := sender.XixunSender{CheckAgent: true}
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = xSender.FileUpload(request, deviceId)

	this.ServeJSON()
}

/*
播放，播放zip文件、MP4播放素材、rtsp视频流
业务1：制定MP4播放素材列表，并点播 (待定)
业务2：查看内部存储里面zip文件是否存在，不存在则调用文件下发，然后再发起播放
*/
func (this *XiXunLedController) LedPlay() {
	deviceId := this.Ctx.Input.Param(":id")
	data := this.Ctx.Input.RequestBody

	xSender := sender.XixunSender{CheckAgent: true}
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = xSender.LedPlay(request, deviceId)
	this.ServeJSON()
}

/*获取本机的消息*/
func (this *XiXunLedController) MsgPublish() {
	deviceId := this.Ctx.Input.Param(":id")

	data := this.Ctx.Input.RequestBody
	xSender := sender.XixunSender{CheckAgent: true}
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = xSender.MsgPublish(request, deviceId)

	this.ServeJSON()
}

/*清除本机的消息*/
func (this *XiXunLedController) Clear() {
	deviceId := this.Ctx.Input.Param(":id")
	xSender := sender.XixunSender{CheckAgent: true}
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL()}
	this.Data["json"] = xSender.ClearScreenText(request, deviceId)
	this.ServeJSON()
}
