package north

import (
	"go-iot/models"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/north/plan/v1",
		beego.NSRouter("/list", &PlanController{}, "post:List"),
		beego.NSRouter("/add", &PlanController{}, "post:Add"),
		beego.NSRouter("/update", &PlanController{}, "post:Update"),
		beego.AddNamespace(ns))
}

type PlanController struct {
	beego.Controller
}

// 查询列表
func (this *PlanController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	this.Data["json"] = models.JsonResp{Success: false, Msg: ""}
	this.ServeJSON()
}

// 添加
func (this *PlanController) Add() {
	// var ob agent.Agent
	// json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	// var resp models.JsonResp
	// resp.Success = true
	// defer func() {
	// 	this.Data["json"] = &resp
	// 	this.ServeJSON()
	// }()
	// var err error

	// resp.Msg = "添加成功!"
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
