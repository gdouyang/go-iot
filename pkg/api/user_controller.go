package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	user "go-iot/pkg/models/base"
	"net/http"
	"strconv"
)

var userResource = Resource{
	Id:   "user-mgr",
	Name: "用户管理",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

func init() {

	getUserAndCheckCreateId := func(ctl *AuthController, userId int64) (*user.UserDTO, error) {
		ob, err := user.GetUser(userId)
		if err != nil {
			return nil, err
		}
		if ob.CreateId != ctl.GetCurrentUser().Id {
			return nil, errors.New("data is not you created")
		}
		return ob, nil
	}
	// 分页查询
	web.RegisterAPI("/user/page", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(userResource, QueryAction) {
			return
		}
		var ob models.PageQuery
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}

		res, err := user.PageUser(&ob, ctl.GetCurrentUser().Id)
		if err != nil {
			ctl.RespError(err)
		} else {
			ctl.RespOkData(res)
		}
	})
	// 新增
	web.RegisterAPI("/user", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(userResource, CretaeAction) {
			return
		}
		var ob user.UserDTO
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ob.CreateId = ctl.GetCurrentUser().Id
		err = user.AddUser(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 修改
	web.RegisterAPI("/user", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(userResource, SaveAction) {
			return
		}
		var ob user.UserDTO
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getUserAndCheckCreateId(ctl, ob.Id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		err = user.UpdateUser(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 单个查询
	web.RegisterAPI("/user/{id}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(userResource, QueryAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		u, err := getUserAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		u.Password = ""
		ctl.RespOkData(u)
	})
	// 删除
	web.RegisterAPI("/user/{id}", "DELETE", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(userResource, DeleteAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getUserAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		var ob *models.User = &models.User{
			Id: int64(_id),
		}
		err = user.DeleteUser(ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 启用
	web.RegisterAPI("/user/enable/{id}", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(userResource, SaveAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getUserAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		var ob *models.User = &models.User{
			Id:         int64(_id),
			EnableFlag: true,
		}
		err = user.UpdateUserEnable(ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 禁用
	web.RegisterAPI("/user/disable/{id}", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(userResource, SaveAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getUserAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		var ob *models.User = &models.User{
			Id:         int64(_id),
			EnableFlag: false,
		}
		err = user.UpdateUserEnable(ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})

	RegResource(userResource)
}
