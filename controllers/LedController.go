package controllers

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/led"
	"go-iot/models/operates"

	"github.com/astaxie/beego"
)

// 设备管理
func init() {
	beego.Router("/led/list", &LedController{}, "post:List")
	beego.Router("/led/add", &LedController{}, "post:Add")
	beego.Router("/led/update", &LedController{}, "post:Update")
	beego.Router("/led/delete", &LedController{}, "post:Delete")
	beego.Router("/led/listProvider", &LedController{}, "post:ListProvider")
}

type LedController struct {
	beego.Controller
}

// 查询设备列表
func (this *LedController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := led.ListDevice(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加设备
func (this *LedController) Add() {
	var ob led.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	err := led.AddDevie(&ob)

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
func (this *LedController) Update() {
	var ob led.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	err := led.UpdateDevice(&ob)
	var resp models.JsonResp
	resp.Success = true
	resp.Msg = "修改成功!"
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
	this.Data["json"] = &resp
	this.ServeJSON()
}

// 删除设备
func (this *LedController) Delete() {
	var ob led.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	led.DeleteDevice(&ob)
	this.Data["json"] = &ob
	this.ServeJSON()
}

// 查询所有厂商
func (this *LedController) ListProvider() {
	pros := operates.AllProvierId()
	this.Data["json"] = &pros
	this.ServeJSON()
}
