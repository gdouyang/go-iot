package api

import (
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	alarm "go-iot/pkg/models/rule"
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
	web.RegisterAPI("/alarm/log/page", "POST", &AlarmController{}, "PageAlarmLog")
	web.RegisterAPI("/alarm/log/{id}/solve", "PUT", &AlarmController{}, "SolveAlarmLog")

	RegResource(alarmResource)
}

type AlarmController struct {
	AuthController
}

func (ctl *AlarmController) PageAlarmLog() {
	if ctl.isForbidden(alarmResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	res, err := alarm.PageAlarmLog(&ob, ctl.GetCurrentUser().Id)
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
