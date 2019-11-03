package xixuncontroller

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/led"
	"go-iot/models/material"
	"go-iot/provider/xixun"
	"go-iot/provider/xixun/sender"
	"strconv"
	"strings"

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

	this.Data["json"] = xSender.ScreenShot(deviceId)

	this.ServeJSON()
}

// LED播放文件上传
func (this *XiXunLedController) FileUpload() {
	deviceId := this.Ctx.Input.Param(":id")
	beego.Info("deviceId=", deviceId)
	var param map[string]string
	json.Unmarshal(this.Ctx.Input.RequestBody, &param)

	device, err := led.GetDevice(deviceId)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		ids := param["ids"]
		materialIds := strings.Split(ids, ",")
		serverUrl := param["serverUrl"]
		serverUrl += "/file/"
		msg := ""
		for _, id := range materialIds {
			material, err := material.GetMaterialById(id)
			if err == nil {
				filename := material.Path
				index := strings.LastIndex(material.Path, "/")
				if index != -1 {
					filename = filename[index+1:]
				}
				operResp := xixun.ProviderImplXiXunLed.FileUpload(device.Sn, serverUrl+material.Path, filename)
				msg += operResp.Msg
			} else {
				msg += err.Error()
			}
		}
		this.Data["json"] = models.JsonResp{Success: true, Msg: msg}
	}

	this.ServeJSON()
}

/*
播放，播放zip文件、MP4播放素材、rtsp视频流
业务1：制定MP4播放素材列表，并点播 (待定)
业务2：查看内部存储里面zip文件是否存在，不存在则调用文件下发，然后再发起播放
*/
func (this *XiXunLedController) LedPlay() {
	deviceId := this.Ctx.Input.Param(":id")
	beego.Info("deviceId=", deviceId)
	var param map[string]string
	json.Unmarshal(this.Ctx.Input.RequestBody, &param)

	device, err := led.GetDevice(deviceId)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		ids := param["ids"]
		serverUrl := param["serverUrl"]
		serverUrl += "/file/"
		material, err := material.GetMaterialById(ids)
		if err == nil {
			filename := material.Path
			index := strings.LastIndex(material.Path, "/")
			if index != -1 {
				filename = filename[index+1:]
			}
			// 查看文件长度，并与远程对比
			length, err := strconv.ParseInt(material.Size, 10, 64)
			if err != nil {
				beego.Error(err)
			}
			beego.Info(filename)
			leg, err := xixun.ProviderImplXiXunLed.FileLength(filename, device.Sn)
			if err != nil {
				this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
				this.ServeJSON()
			}
			if length != leg {
				//长度不一致，则返回，让重新上传
				this.Data["json"] = models.JsonResp{Success: false, Msg: "文件长度不一致，文件没有上传成功"}
				this.ServeJSON()
			}
			//如果长度一致，就发起播放
			operResp := xixun.ProviderImplXiXunLed.PlayZip(filename, device.Sn)
			this.Data["json"] = models.JsonResp{Success: operResp.Success, Msg: operResp.Msg}
		}
	}
	this.ServeJSON()
}

/*获取本机的消息*/
func (this *XiXunLedController) MsgPublish() {
	deviceId := this.Ctx.Input.Param(":id")

	data := this.Ctx.Input.RequestBody
	xSender := sender.XixunSender{CheckAgent: true}
	this.Data["json"] = xSender.MsgPublish(data, deviceId)

	this.ServeJSON()
}

/*清除本机的消息*/
func (this *XiXunLedController) Clear() {
	deviceId := this.Ctx.Input.Param(":id")
	xSender := sender.XixunSender{CheckAgent: true}
	this.Data["json"] = xSender.ClearScreenText(deviceId)
	this.ServeJSON()
}
