package api

import (
	"go-iot/models"
	"go-iot/models/notify"
	notify1 "go-iot/notify"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

var notifyResource = Resource{
	Id:   "notify-config",
	Name: "通知配置",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

func init() {
	ns := web.NewNamespace("/api/notifier/config",
		web.NSRouter("/page", &NotifyController{}, "post:List"),
		web.NSRouter("/list", &NotifyController{}, "post:ListAll"),
		web.NSRouter("/", &NotifyController{}, "post:Add"),
		web.NSRouter("/:id", &NotifyController{}, "put:Update"),
		web.NSRouter("/:id", &NotifyController{}, "get:Get"),
		web.NSRouter("/types", &NotifyController{}, "get:Types"),
		web.NSRouter("/:id", &NotifyController{}, "delete:Delete"),
		web.NSRouter("/:id/start", &NotifyController{}, "post:Enable"),
		web.NSRouter("/:id/stop", &NotifyController{}, "post:Disable"),
	)
	web.AddNamespace(ns)

	regResource(notifyResource)
}

type NotifyController struct {
	AuthController
}

func (ctl *NotifyController) List() {
	if ctl.isForbidden(notifyResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := notify.ListNotify(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {
		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
}

func (ctl *NotifyController) ListAll() {
	if ctl.isForbidden(notifyResource, QueryAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Notify
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	res, err := notify.ListAll(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
	} else {
		resp = models.JsonRespOkData(res)
	}
}

func (ctl *NotifyController) Get() {
	if ctl.isForbidden(notifyResource, QueryAction) {
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
	u, err := notify.GetNotifyMust(int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOkData(u)
}

func (ctl *NotifyController) Types() {
	if ctl.isForbidden(notifyResource, QueryAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	list := notify1.GetAllNotify()
	var list1 []map[string]interface{}
	for _, v := range list {
		list1 = append(list1, map[string]interface{}{
			"type":   v.Kind(),
			"name":   v.Name(),
			"config": v.Meta(),
		})
	}
	resp = models.JsonRespOkData(list1)
}

func (ctl *NotifyController) Add() {
	if ctl.isForbidden(notifyResource, CretaeAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Notify
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = notify.AddNotify(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *NotifyController) Update() {
	if ctl.isForbidden(notifyResource, SaveAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Notify
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = notify.UpdateNotify(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *NotifyController) Delete() {
	if ctl.isForbidden(notifyResource, DeleteAction) {
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
	var ob *models.Notify = &models.Notify{
		Id: int64(_id),
	}
	err = notify.DeleteNotify(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *NotifyController) Enable() {
	if ctl.isForbidden(notifyResource, SaveAction) {
		return
	}
	ctl.enable(true)
}

func (ctl *NotifyController) Disable() {
	if ctl.isForbidden(notifyResource, SaveAction) {
		return
	}
	ctl.enable(false)
}

func (ctl *NotifyController) enable(flag bool) {
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
	m, err := notify.GetNotifyMust(int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	var state string = models.Started
	if flag {
		config := notify1.NotifyConfig{Config: m.Config, Template: m.Template}
		err = notify1.EnableNotify(m.Type, m.Id, config)
		if err != nil {
			resp = models.JsonRespError(err)
			return
		}
	} else {
		state = models.Stopped
		notify1.DisableNotify(m.Id)
	}
	err = notify.UpdateNotifyState(&models.Notify{
		Id:    m.Id,
		State: state,
	})
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}