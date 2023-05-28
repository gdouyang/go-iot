package api

import (
	"errors"
	"go-iot/pkg/models"
	"go-iot/pkg/models/notify"
	notify1 "go-iot/pkg/notify"
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
		web.NSRouter("/page", &NotifyController{}, "post:Page"),
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

func (ctl *NotifyController) Page() {
	if ctl.isForbidden(notifyResource, QueryAction) {
		return
	}
	var ob models.PageQuery[models.Notify]
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := notify.PageNotify(&ob, ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (ctl *NotifyController) ListAll() {
	if ctl.isForbidden(notifyResource, QueryAction) {
		return
	}
	var ob models.Notify
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.CreateId = ctl.GetCurrentUser().Id
	res, err := notify.ListAll(&ob, &ob.CreateId)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (ctl *NotifyController) Get() {
	if ctl.isForbidden(notifyResource, QueryAction) {
		return
	}
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	u, err := ctl.getNotifyAndCheckCreateId(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(u)
}

func (ctl *NotifyController) Types() {
	if ctl.isForbidden(notifyResource, QueryAction) {
		return
	}
	list := notify1.GetAllNotify()
	var list1 []map[string]interface{}
	for _, v := range list {
		list1 = append(list1, map[string]interface{}{
			"type":   v.Kind(),
			"name":   v.Name(),
			"config": v.Meta(),
		})
	}
	ctl.RespOkData(list1)
}

func (ctl *NotifyController) Add() {
	if ctl.isForbidden(notifyResource, CretaeAction) {
		return
	}
	var ob models.Notify
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.CreateId = ctl.GetCurrentUser().Id
	err = notify.AddNotify(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *NotifyController) Update() {
	if ctl.isForbidden(notifyResource, SaveAction) {
		return
	}
	var ob models.Notify
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	_, err = ctl.getNotifyAndCheckCreateId(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = notify.UpdateNotify(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *NotifyController) Delete() {
	if ctl.isForbidden(notifyResource, DeleteAction) {
		return
	}
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	_, err = ctl.getNotifyAndCheckCreateId(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	var ob *models.Notify = &models.Notify{
		Id: int64(_id),
	}
	err = notify.DeleteNotify(ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
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
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	m, err := ctl.getNotifyAndCheckCreateId(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	var state string = models.Started
	if flag {
		config := notify1.NotifyConfig{Name: m.Name, Config: m.Config, Template: m.Template}
		err = notify1.EnableNotify(m.Type, m.Id, config)
		if err != nil {
			ctl.RespError(err)
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
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *NotifyController) getNotifyAndCheckCreateId(id int64) (*models.Notify, error) {
	ob, err := notify.GetNotifyMust(id)
	if err != nil {
		return nil, err
	}
	if ob.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New("data is not you created")
	}
	return ob, nil
}
