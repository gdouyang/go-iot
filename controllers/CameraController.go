package controllers

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/camera"
	"strconv"

	"github.com/astaxie/beego"
)

// 设备管理
func init() {
	beego.Router("/camera/list", &CameraController{}, "post:List")
	beego.Router("/camera/add", &CameraController{}, "post:Add")
	beego.Router("/camera/update", &CameraController{}, "post:Update")
	beego.Router("/camera/delete", &CameraController{}, "post:Delete")
	beego.Router("/camera/detail/:id", &CameraController{}, "get:GetCamera")
}

type CameraController struct {
	beego.Controller
}

// 查询设备列表
func (this *CameraController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	res, err := camera.ListCamera(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加设备
func (this *CameraController) Add() {
	data := this.Ctx.Input.RequestBody
	ob := camera.Camera{}
	json.Unmarshal(data, &ob)
	//add
	err := camera.AddCamera(&ob)
	this.Data["json"] = models.JsonResp{Success: true, Msg: "添加成功"}
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	}
	this.ServeJSON()
}

// 更新设备信息
func (this *CameraController) Update() {
	data := this.Ctx.Input.RequestBody
	ob := camera.Camera{}
	json.Unmarshal(data, &ob)

	this.Data["json"] = camera.UpdateCamera(&ob)
	this.ServeJSON()
}

// 删除设备
func (this *CameraController) Delete() {
	data := this.Ctx.Input.RequestBody
	ob := camera.Camera{}
	json.Unmarshal(data, &ob)

	this.Data["json"] = camera.DeleteCamera(&ob)
	this.ServeJSON()
}

// 获取设备
func (this *CameraController) GetCamera() {
	id := this.Ctx.Input.Param(":id")
	cid, err := strconv.Atoi(id)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: "无效查询"}
		this.ServeJSON()
	}
	ob, err := camera.GetCameraByID(cid)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		this.Data["json"] = &ob
	}
	this.ServeJSON()

}
