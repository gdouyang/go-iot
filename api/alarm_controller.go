package api

import (
	"go-iot/models"
	alarm "go-iot/models/scene"

	"github.com/beego/beego/v2/server/web"
)

var alarmResource = Resource{
	Id:   "alarm-mgr",
	Name: "设备告警",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

func init() {
	ns := web.NewNamespace("/api/alarm",
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
