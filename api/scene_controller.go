package api

import (
	"go-iot/models"
	"go-iot/models/scene"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

var sceneResource = Resource{
	Id:   "rule-mgr",
	Name: "场景联动",
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
		web.NSRouter("/", &SceneController{}, "put:Update"),
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
	s := models.SceneModel{}
	s.FromEnitty(*u)
	resp = models.JsonRespOkData(s)
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
	en := ob.ToEnitty()
	err = scene.AddScene(&en)
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
	en := ob.ToEnitty()
	err = scene.UpdateScene(&en)
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
	err = scene.UpdateSceneStatus(models.Started, int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *SceneController) Disable() {
	if ctl.isForbidden(sceneResource, SaveAction) {
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
	err = scene.UpdateSceneStatus(models.Stopped, int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}
