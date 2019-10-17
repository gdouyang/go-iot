package xixun

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/material"
	"strings"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/control/xixun/v1",
		beego.NSRouter("/:id/screenShot", &XiXunLedController{}, "post:ScreenShot"),
		beego.NSRouter("/:id/fileUpload", &XiXunLedController{}, "post:FileUpload"))
	beego.AddNamespace(ns)
}

type XiXunLedController struct {
	beego.Controller
}

// LED截图
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

// LED播放文件上传
func (this *XiXunLedController) FileUpload() {
	deviceId := this.Ctx.Input.Param(":id")
	beego.Info("deviceId=", deviceId)
	var param map[string]string
	json.Unmarshal(this.Ctx.Input.RequestBody, &param)

	device, err := models.GetDevice(deviceId)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		ids := param["ids"]
		materialIds := strings.Split(ids, ",")
		serverUrl := param["serverUrl"]
		serverUrl += "/file/"
		for _, id := range materialIds {
			material, err := material.GetMaterialById(id)
			if err == nil {
				operResp := ProviderImplXiXunLed.FileUpload(device.Sn, serverUrl+material.Id, material.Name)
				this.Data["json"] = models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
			}
		}
	}

	this.ServeJSON()
}
