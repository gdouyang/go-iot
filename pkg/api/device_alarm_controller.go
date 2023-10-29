package api

import (
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	alarm "go-iot/pkg/models/rule"
	"net/http"
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
	// 告警分页接口
	web.RegisterAPI("/alarm/log/page", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	// 处理告警
	web.RegisterAPI("/alarm/log/{id}/solve", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})

	RegResource(alarmResource)
}
