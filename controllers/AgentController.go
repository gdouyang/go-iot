package controllers

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/agent"
	"net/http"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/agent/list", &AgentController{}, "post:List")
	beego.Router("/agent/add", &AgentController{}, "post:Add")
	beego.Router("/agent/update", &AgentController{}, "post:Update")
	beego.Router("/agent/delete", &AgentController{}, "post:Delete")
	beego.Router("/agent/get/:id", &AgentController{}, "post:Get")
}

type AgentController struct {
	beego.Controller
}

// 查询列表
func (this *AgentController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := agent.ListAgent(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加
func (this *AgentController) Add() {
	var ob agent.Agent
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	var resp models.JsonResp
	resp.Success = true
	defer func() {
		this.Data["json"] = &resp
		this.ServeJSON()
	}()
	var err error

	resp.Msg = "添加成功!"
	// 保存入库
	err = agent.AddAgent(&ob)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
}
func (this *AgentController) Update() {
	var ob agent.Agent
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	var resp models.JsonResp
	resp.Success = true
	defer func() {
		this.Data["json"] = &resp
		this.ServeJSON()
	}()
	var err error
	resp.Msg = "修改成功!"
	// 保存入库
	err = agent.UpdateAgent(&ob)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
}

// 删除
func (this *AgentController) Delete() {
	var ob agent.Agent
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	agent.DeleteAgent(&ob)
	this.Data["json"] = &ob
	this.ServeJSON()
}

func (this *AgentController) Get() {
	agentId := this.Ctx.Input.Param(":id")
	agent, err := agent.GetAgent(agentId)
	if err != nil {
		http.Error(this.Ctx.ResponseWriter, "Agent Not found", 404)
		return
	}
	this.Data["json"] = &agent
	this.ServeJSON()
}
