package north

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/plan"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/plan",
		beego.NSRouter("/list", &PlanController{}, "post:List"),
		beego.NSRouter("/add", &PlanController{}, "post:Add"),
		beego.NSRouter("/delete", &PlanController{}, "post:Delete"),
		beego.NSRouter("/update", &PlanController{}, "post:Update"))
	beego.AddNamespace(ns)
}

type PlanController struct {
	beego.Controller
}

// 查询列表
func (this *PlanController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	rest, err := plan.ListPlan(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: ""}
	} else {
		this.Data["json"] = rest
	}
	this.ServeJSON()
}

// 添加
func (this *PlanController) Add() {
	var ob plan.Plan
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	var resp models.JsonResp
	resp.Success = true
	defer func() {
		this.Data["json"] = &resp
		this.ServeJSON()
	}()
	err := plan.AddPlan(&ob)
	resp.Msg = "添加成功!"
	if err != nil {
		resp.Msg = err.Error()
	}
}

// 删除
func (this *PlanController) Delete() {
	data := this.Ctx.Input.RequestBody
	var ob plan.Plan
	json.Unmarshal(data, &ob)

	var resp models.JsonResp
	resp.Success = true

	err := plan.DeletePlan(&ob)
	resp.Msg = "删除成功!"
	if err != nil {
		resp.Msg = err.Error()
	}
	this.Data["json"] = resp
	this.ServeJSON()
}

func (this *PlanController) Update() {
	// var ob agent.Agent
	// json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	// var resp models.JsonResp
	// resp.Success = true
	// defer func() {
	// 	this.Data["json"] = &resp
	// 	this.ServeJSON()
	// }()
	// var err error
	// resp.Msg = "修改成功!"
	// // 保存入库
	// err = agent.UpdateAgent(&ob)
	// if err != nil {
	// 	resp.Msg = err.Error()
	// 	resp.Success = false
	// }
}
