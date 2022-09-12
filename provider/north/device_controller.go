package north

import (
	"encoding/json"
	"go-iot/models"
	device "go-iot/models/device"

	"github.com/beego/beego/v2/server/web"
)

// 设备管理
func init() {
	web.Router("/led/list", &DeviceController{}, "post:List")
	web.Router("/north/led/add", &DeviceController{}, "post:Add")
	web.Router("/north/led/update", &DeviceController{}, "post:Update")
	web.Router("/north/led/delete", &DeviceController{}, "post:Delete")

}

var ()

type DeviceController struct {
	web.Controller
}

// 查询设备列表
func (ctl *DeviceController) List() {
	var ob models.PageQuery
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)

	res, err := device.ListDevice(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		ctl.Data["json"] = &res
	}
	ctl.ServeJSON()
}

// 添加设备
func (ctl *DeviceController) Add() {
	var ob models.Device
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	ctl.Data["json"] = device.AddDevice(&ob)
	ctl.ServeJSON()
}

// 更新设备信息
func (ctl *DeviceController) Update() {
	var ob models.Device
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	ctl.Data["json"] = device.UpdateDevice(&ob)
	ctl.ServeJSON()
}

// 删除设备
func (ctl *DeviceController) Delete() {
	var ob models.Device
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	ctl.Data["json"] = device.DeleteDevice(&ob)
	ctl.ServeJSON()
}
