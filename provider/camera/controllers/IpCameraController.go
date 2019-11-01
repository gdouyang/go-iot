package xixuncontroller

import (
	"encoding/json"
	"go-iot/agent"
	"go-iot/models"
	"go-iot/models/led"
	"go-iot/models/material"
	"go-iot/models/operates"
	"go-iot/provider/xixun"
	"go-iot/provider/xixun/sender"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control/camera/v1",
		beego.NSRouter("/:id/play", &IpCameraController{}, "post:Play")
		)
	beego.AddNamespace(ns)
}

type IpCameraController struct {
	beego.Controller
}

// 播放
func (this *IpCameraController) Play() {
	deviceId := this.Ctx.Input.Param(":id")

	byteReq := []byte("{}")
	xSender := sender.XixunSender{CheckAgent: true}
	xSender.SendAgent = func(device operates.Device) models.JsonResp {
		req := agent.NewRequest(device.Id, device.Sn, device.Provider, sender.SCREEN_SHOT, byteReq)
		res := agent.SendCommand(device.Agent, req)
		return res
	}

	this.Data["json"] = xSender.ScreenShot(deviceId)

	this.ServeJSON()
}
