package api

import (
	"go-iot/models"
	alarm "go-iot/models/scene"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

var alarmResource = Resource{
	Id:   "alarm-mgr",
	Name: "告警管理",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

func init() {
	ns := web.NewNamespace("/api/alarm",
		web.NSRouter("/page", &AlarmController{}, "post:List"),
		web.NSRouter("/", &AlarmController{}, "post:Add"),
		web.NSRouter("/:id", &AlarmController{}, "put:Update"),
		web.NSRouter("/:id", &AlarmController{}, "get:Get"),
		web.NSRouter("/:id", &AlarmController{}, "delete:Delete"),
		web.NSRouter("/:id/start", &AlarmController{}, "put:Enable"),
		web.NSRouter("/:id/stop", &AlarmController{}, "put:Disable"),
		web.NSRouter("/:target/:targetId", &AlarmController{}, "put:GetAlarmList"),
		web.NSRouter("/log/page", &AlarmController{}, "post:PageAlarmLog"),
		web.NSRouter("/log/:id/solve", &AlarmController{}, "put:SolveAlarmLog"),
	)
	web.AddNamespace(ns)

	regResource(alarmResource)
}

type AlarmController struct {
	AuthController
}

func (ctl *AlarmController) List() {
	if ctl.isForbidden(alarmResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := alarm.ListAlarm(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {
		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
}

func (ctl *AlarmController) Get() {
	if ctl.isForbidden(alarmResource, QueryAction) {
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
	u, err := alarm.GetAlarm(int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOkData(u)
}

func (ctl *AlarmController) Add() {
	if ctl.isForbidden(alarmResource, CretaeAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Alarm
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}

	err = alarm.AddAlarm(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *AlarmController) Update() {
	if ctl.isForbidden(alarmResource, SaveAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Alarm
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = alarm.UpdateAlarm(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *AlarmController) Delete() {
	if ctl.isForbidden(alarmResource, DeleteAction) {
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
	var ob *models.Alarm = &models.Alarm{
		Id: int64(_id),
	}
	err = alarm.DeleteAlarm(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *AlarmController) Enable() {
	if ctl.isForbidden(alarmResource, SaveAction) {
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
	err = alarm.UpdateAlarmStatus(models.Started, int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *AlarmController) Disable() {
	if ctl.isForbidden(alarmResource, SaveAction) {
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
	err = alarm.UpdateAlarmStatus(models.Stopped, int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *AlarmController) GetAlarmList() {
	if ctl.isForbidden(alarmResource, QueryAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	target := ctl.Ctx.Input.Param(":target")
	targetId := ctl.Ctx.Input.Param(":targetId")
	var q = models.Alarm{
		Target:   target,
		TargetId: targetId,
	}
	list, err := alarm.GetAlarmList(q)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp.Data = list
}

func (ctl *AlarmController) GetAlarmLog() {
	if ctl.isForbidden(alarmResource, QueryAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	target := ctl.Ctx.Input.Param(":target")
	targetId := ctl.Ctx.Input.Param(":targetId")
	var q models.AlarmLog
	if target == "device" {
		q.DeviceId = targetId
	} else {
		q.ProductId = targetId
	}
	list, err := alarm.GetAlarmLog(q)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp.Data = list
}

func (ctl *AlarmController) PageAlarmLog() {
	if ctl.isForbidden(alarmResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := alarm.PageAlarmLog(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {
		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
}

func (ctl *AlarmController) SolveAlarmLog() {
	if ctl.isForbidden(alarmResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	var ob models.AlarmLog
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = alarm.SolveAlarmLog(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
}
