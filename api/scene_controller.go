package api

import (
	"go-iot/models"
	"go-iot/models/scene"
	"go-iot/ruleengine"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

var sceneResource = Resource{
	Id:   "rule-mgr",
	Name: "规则引擎",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

func init() {
	ns := web.NewNamespace("/api/scene",
		web.NSRouter("/page", &SceneController{}, "post:List"),
		web.NSRouter("/", &SceneController{}, "post:Add"),
		web.NSRouter("/:id", &SceneController{}, "put:Update"),
		web.NSRouter("/:id", &SceneController{}, "get:Get"),
		web.NSRouter("/:id", &SceneController{}, "delete:Delete"),
		web.NSRouter("/:id/start", &SceneController{}, "post:Enable"),
		web.NSRouter("/:id/stop", &SceneController{}, "post:Disable"),
	)
	web.AddNamespace(ns)

	regResource(sceneResource)
}

type SceneController struct {
	AuthController
}

func (ctl *SceneController) List() {
	if ctl.isForbidden(sceneResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := scene.ListScene(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {
		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
}

func (ctl *SceneController) Get() {
	if ctl.isForbidden(sceneResource, QueryAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	id := ctl.Ctx.Input.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	u, err := scene.GetSceneMust(int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOkData(u)
}

func (ctl *SceneController) Add() {
	if ctl.isForbidden(sceneResource, CretaeAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.SceneModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = scene.AddScene(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *SceneController) Update() {
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.SceneModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = scene.UpdateScene(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *SceneController) Delete() {
	if ctl.isForbidden(sceneResource, DeleteAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	id := ctl.Ctx.Input.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		return
	}
	var ob *models.Scene = &models.Scene{
		Id: int64(_id),
	}
	err = scene.DeleteScene(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *SceneController) Enable() {
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	ctl.enable(true)
}

func (ctl *SceneController) Disable() {
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	ctl.enable(false)
}

func (ctl *SceneController) enable(flag bool) {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	id := ctl.Ctx.Input.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	m, err := scene.GetSceneMust(int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	var state string = models.Started
	if flag {
		rule := ruleengine.RuleExecutor{
			ProductId:   m.ProductId,
			TriggerType: ruleengine.TriggerType(m.TriggerType),
			Cron:        m.Cron,
			Trigger:     m.Trigger,
			Actions:     m.Actions,
			DeviceIds:   m.DeviceIds,
		}
		err = ruleengine.StartScene(m.Id, rule)
		if err != nil {
			resp = models.JsonRespError(err)
			return
		}
	} else {
		state = models.Stopped
		ruleengine.StopScene(m.Id)
	}

	err = scene.UpdateSceneStatus(state, m.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}
