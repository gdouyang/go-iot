package api

import (
	"go-iot/models"
	alarm "go-iot/models/rule"

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
		web.NSRouter("/log/page", &AlarmController{}, "post:PageAlarmLog"),
		web.NSRouter("/log/:id/solve", &AlarmController{}, "put:SolveAlarmLog"),
	)
	web.AddNamespace(ns)

	regResource(alarmResource)
}

type AlarmController struct {
	AuthController
}

func (ctl *AlarmController) PageAlarmLog() {
	if ctl.isForbidden(alarmResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := alarm.PageAlarmLog(&ob)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (ctl *AlarmController) SolveAlarmLog() {
	if ctl.isForbidden(alarmResource, SaveAction) {
		return
	}
	var ob models.AlarmLog
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = alarm.SolveAlarmLog(ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}
