package north

import (
	"encoding/json"
	"go-iot/models"
	led "go-iot/models/device"
	"go-iot/provider/north/sender"

	"github.com/beego/beego/v2/server/web"
)

// 设备管理
func init() {
	web.Router("/led/list", &LedController{}, "post:List")
	web.Router("/north/led/add", &LedController{}, "post:Add")
	web.Router("/north/led/update", &LedController{}, "post:Update")
	web.Router("/north/led/delete", &LedController{}, "post:Delete")

}

var (
	ledSender sender.LedSender = sender.LedSender{CheckAgent: true}
)

type LedController struct {
	web.Controller
}

// 查询设备列表
func (ctl *LedController) List() {
	var ob models.PageQuery
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)

	res, err := led.ListDevice(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		ctl.Data["json"] = &res
	}
	ctl.ServeJSON()
}

// 添加设备
func (ctl *LedController) Add() {
	data := ctl.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Url: ctl.Ctx.Input.URL(), Data: data}
	ctl.Data["json"] = ledSender.Add(request)
	ctl.ServeJSON()
}

// 更新设备信息
func (ctl *LedController) Update() {
	data := ctl.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Url: ctl.Ctx.Input.URL(), Data: data}
	ctl.Data["json"] = ledSender.Update(request)
	ctl.ServeJSON()
}

// 删除设备
func (ctl *LedController) Delete() {
	data := ctl.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Url: ctl.Ctx.Input.URL(), Data: data}
	ctl.Data["json"] = ledSender.Delete(request)
	ctl.ServeJSON()
}
