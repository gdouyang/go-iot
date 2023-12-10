package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	dao "go-iot/pkg/models/notify"
	"go-iot/pkg/notify"
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

	a := &notifyApi{}
	web.RegisterAPI("/notifier/config/page", "POST", a.page)
	web.RegisterAPI("/notifier/config/list", "POST", a.list)
	// 新增
	web.RegisterAPI("/notifier/config", "POST", a.add)
	// 修改
	web.RegisterAPI("/notifier/config/{id}", "PUT", a.update)
	web.RegisterAPI("/notifier/config/{id}", "GET", a.get)
	// 查询通知类型
	web.RegisterAPI("/notifier/config/types", "GET", a.getTypes)
	// 删除通知配置
	web.RegisterAPI("/notifier/config/{id}", "DELETE", a.delete)
	// 复制通知
	web.RegisterAPI("/notifier/config/{id}/copy", "POST", a.copy)
	// 测试
	web.RegisterAPI("/notifier/config/test", "POST", a.test)
	// 启动
	web.RegisterAPI("/notifier/config/{id}/start", "POST", a.start)
	// 停用
	web.RegisterAPI("/notifier/config/{id}/stop", "POST", a.stop)
}

var defaultNotifyApi notifyApi

type notifyApi struct {
}

func (a *notifyApi) page(w http.ResponseWriter, r *http.Request) {
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

	res, err := dao.PageNotify(&ob, &ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (a *notifyApi) list(w http.ResponseWriter, r *http.Request) {
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
	res, err := dao.ListAll(&ob, &ob.CreateId)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (a *notifyApi) add(w http.ResponseWriter, r *http.Request) {
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
	err = dao.AddNotify(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (a *notifyApi) update(w http.ResponseWriter, r *http.Request) {
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
	_, err = defaultNotifyApi.getNotifyAndCheckCreateId(ctl, ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = dao.UpdateNotify(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (a *notifyApi) get(w http.ResponseWriter, r *http.Request) {
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
	u, err := defaultNotifyApi.getNotifyAndCheckCreateId(ctl, int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(u)
}

func (a *notifyApi) getTypes(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(notifyResource, QueryAction) {
		return
	}
	list := notify.GetAllNotify()
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

func (a *notifyApi) delete(w http.ResponseWriter, r *http.Request) {
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
	_, err = defaultNotifyApi.getNotifyAndCheckCreateId(ctl, int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	var ob *models.Notify = &models.Notify{
		Id: int64(_id),
	}
	err = dao.DeleteNotify(ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (a *notifyApi) copy(w http.ResponseWriter, r *http.Request) {
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
	_, err = defaultNotifyApi.getNotifyAndCheckCreateId(ctl, int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	var ob *models.Notify = &models.Notify{
		Id: int64(_id),
	}
	data, err := dao.GetNotify(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if data == nil {
		ctl.RespError(errors.New("数据不存在"))
		return
	}
	data.Id = 0
	dao.AddNotify(data)
	ctl.RespOk()
}

func (a *notifyApi) test(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(notifyResource, SaveAction) {
		return
	}
	var m models.Notify
	err := ctl.BindJSON(&m)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = notify.TestNotify(m.Type, notify.NotifyConfig{Name: m.Name, Config: m.Config, Template: m.Template})
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (a *notifyApi) start(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(notifyResource, SaveAction) {
		return
	}
	defaultNotifyApi.enableNotify(ctl, true)
}

func (a *notifyApi) stop(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(notifyResource, SaveAction) {
		return
	}
	defaultNotifyApi.enableNotify(ctl, false)
}

func (a *notifyApi) getNotifyAndCheckCreateId(ctl *AuthController, id int64) (*models.Notify, error) {
	ob, err := dao.GetNotifyMust(id)
	if err != nil {
		return nil, err
	}
	if ob.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New("data is not you created")
	}
	return ob, nil
}

func (a *notifyApi) enableNotify(ctl *AuthController, flag bool) {
	id := ctl.Param("id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	m, err := defaultNotifyApi.getNotifyAndCheckCreateId(ctl, int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	var state string = models.Started
	if flag {
		err = notify.EnableNotify(m.Type, m.Id, notify.NotifyConfig{Name: m.Name, Config: m.Config, Template: m.Template})
		if err != nil {
			ctl.RespError(err)
			return
		}
	} else {
		state = models.Stopped
		notify.DisableNotify(m.Id)
	}
	err = dao.UpdateNotifyState(&models.Notify{
		Id:    m.Id,
		State: state,
	})
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}
