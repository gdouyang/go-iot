package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	"go-iot/pkg/models/notify"
	notify1 "go-iot/pkg/notify"
	"net/http"
	"strconv"
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
	RegResource(notifyResource)

	getNotifyAndCheckCreateId := func(ctl *AuthController, id int64) (*models.Notify, error) {
		ob, err := notify.GetNotifyMust(id)
		if err != nil {
			return nil, err
		}
		if ob.CreateId != ctl.GetCurrentUser().Id {
			return nil, errors.New("data is not you created")
		}
		return ob, nil
	}
	web.RegisterAPI("/notifier/config/page", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(notifyResource, QueryAction) {
			return
		}
		var ob models.PageQuery
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}

		res, err := notify.PageNotify(&ob, &ctl.GetCurrentUser().Id)
		if err != nil {
			ctl.RespError(err)
		} else {
			ctl.RespOkData(res)
		}
	})
	web.RegisterAPI("/notifier/config/list", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	// 新增
	web.RegisterAPI("/notifier/config", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	// 修改
	web.RegisterAPI("/notifier/config/{id}", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(notifyResource, SaveAction) {
			return
		}
		var ob models.Notify
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getNotifyAndCheckCreateId(ctl, ob.Id)
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
	})
	web.RegisterAPI("/notifier/config/{id}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(notifyResource, QueryAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		u, err := getNotifyAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOkData(u)
	})
	// 查询通知类型
	web.RegisterAPI("/notifier/config/types", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	// 删除通知配置
	web.RegisterAPI("/notifier/config/{id}", "DELETE", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(notifyResource, DeleteAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getNotifyAndCheckCreateId(ctl, int64(_id))
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
	})
	// 复制通知
	web.RegisterAPI("/notifier/config/{id}/copy", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(notifyResource, CretaeAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getNotifyAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		var ob *models.Notify = &models.Notify{
			Id: int64(_id),
		}
		data, err := notify.GetNotify(ob.Id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if data == nil {
			ctl.RespError(errors.New("data not found"))
			return
		}
		data.Id = 0
		notify.AddNotify(data)
		ctl.RespOk()
	})
	enableNotify := func(ctl *AuthController, flag bool) {
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		m, err := getNotifyAndCheckCreateId(ctl, int64(_id))
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
	// 启动
	web.RegisterAPI("/notifier/config/{id}/start", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(notifyResource, SaveAction) {
			return
		}
		enableNotify(ctl, true)
	})
	// 停用
	web.RegisterAPI("/notifier/config/{id}/stop", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(notifyResource, SaveAction) {
			return
		}
		enableNotify(ctl, false)
	})

}
