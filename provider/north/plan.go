package north

import (
	"encoding/json"
	"go-iot/models"

	"go-iot/models/plan"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	ns := web.NewNamespace("/north/plan",
		web.NSRouter("/list", &PlanController{}, "post:List"),
		web.NSRouter("/add", &PlanController{}, "post:Add"),
		web.NSRouter("/delete", &PlanController{}, "post:Delete"),
		web.NSRouter("/update", &PlanController{}, "post:Update"))
	web.AddNamespace(ns)
}

type PlanController struct {
	web.Controller
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
	var ob plan.Plan
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	var resp models.JsonResp
	resp.Success = true
	defer func() {
		this.Data["json"] = &resp
		this.ServeJSON()
	}()
	err := plan.UpdatePlan(&ob)
	resp.Msg = "修改成功!"
	if err != nil {
		resp.Msg = err.Error()
	}
}
