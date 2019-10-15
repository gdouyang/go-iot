package controllers

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/material"

	"github.com/astaxie/beego"
)

// 素材管理
func init() {
	beego.Router("/material/list", &MaterialController{}, "post:List")
	beego.Router("/material/add", &MaterialController{}, "post:Add")
	beego.Router("/material/update", &MaterialController{}, "post:Update")
	beego.Router("/material/delete", &MaterialController{}, "post:Delete")
}

type MaterialController struct {
	beego.Controller
}

// 查询设备列表
func (this *MaterialController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := material.ListMaterial(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加设备
func (this *MaterialController) Add() {
	var ob material.Material
	ob.Name = this.GetString("name")
	ob.Type = this.GetString("type")
	f, h, err := this.GetFile("uploadname")
	defer f.Close()

	var resp models.JsonResp
	resp.Success = true
	resp.Msg = "添加成功!"
	defer func() {
		this.Data["json"] = &resp
		this.ServeJSON()
	}()
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	filePath := "/files/" + h.Filename
	err = this.SaveToFile("uploadname", "."+filePath)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	ob.Path = filePath
	// 保存入库
	err = material.AddMaterial(&ob)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
}

// 更新设备信息
func (this *MaterialController) Update() {
	var ob material.Material
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	err := material.UpdateMaterial(&ob)
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
func (this *MaterialController) Delete() {
	var ob material.Material
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	material.DeleteMaterial(&ob)
	this.Data["json"] = &ob
	this.ServeJSON()
}
