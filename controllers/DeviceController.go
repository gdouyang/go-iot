package controllers

import (
	"encoding/json"
	"go-iot/models"

	"github.com/astaxie/beego"
)

// 设备管理
func init() {
	beego.Router("/device/list", &DeviceController{}, "post:List")
	beego.Router("/device/add", &DeviceController{}, "post:Add")
	beego.Router("/device/update", &DeviceController{}, "post:Update")
	beego.Router("/device/delete", &DeviceController{}, "post:Delete")
}

type DeviceController struct {
	beego.Controller
}

// 查询设备列表
func (this *DeviceController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := models.ListDevice(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加设备
func (this *DeviceController) Add() {
	var ob models.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	err := models.AddDevie(&ob)

	var resp models.JsonResp
	resp.Success = true
	resp.Msg = "添加成功!"
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
	this.Data["json"] = &resp
	this.ServeJSON()
}

// 更新设备信息
func (this *DeviceController) Update() {
	var ob models.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	models.UpdateDevice(&ob)
	this.Data["json"] = &ob
	this.ServeJSON()
}

// 删除设备
func (this *DeviceController) Delete() {
	var ob models.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	models.DeleteDevice(&ob)
	this.Data["json"] = &ob
	this.ServeJSON()
}
