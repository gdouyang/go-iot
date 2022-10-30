package api

import (
	"go-iot/models"
	user "go-iot/models/base"
	"strconv"

	"github.com/beego/beego/v2/server/web"
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
	ns := web.NewNamespace("/api/user",
		web.NSRouter("/page", &UserController{}, "post:List"),
		web.NSRouter("/", &UserController{}, "post:Add"),
		web.NSRouter("/", &UserController{}, "put:Update"),
		web.NSRouter("/:id", &UserController{}, "get:Get"),
		web.NSRouter("/:id", &UserController{}, "delete:Delete"),
		web.NSRouter("/enable/:id", &UserController{}, "put:Enable"),
		web.NSRouter("/disable/:id", &UserController{}, "put:Disable"),
	)
	web.AddNamespace(ns)

	regResource(userResource)
}

type UserController struct {
	AuthController
}

func (ctl *UserController) List() {
	if ctl.isForbidden(userResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := user.ListUser(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {
		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
}

func (ctl *UserController) Get() {
	if ctl.isForbidden(userResource, QueryAction) {
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
	u, err := user.GetUser(int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	u.Password = ""
	resp = models.JsonRespOkData(u)
}

func (ctl *UserController) Add() {
	if ctl.isForbidden(userResource, CretaeAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.User
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}

	err = user.AddUser(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *UserController) Update() {
	if ctl.isForbidden(userResource, SaveAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.User
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = user.UpdateUser(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *UserController) Delete() {
	if ctl.isForbidden(userResource, DeleteAction) {
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
	var ob *models.User = &models.User{
		Id: int64(_id),
	}
	err = user.DeleteUser(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *UserController) Enable() {
	if ctl.isForbidden(userResource, SaveAction) {
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
	var ob *models.User = &models.User{
		Id:         int64(_id),
		EnableFlag: true,
	}
	err = user.UpdateUserEnable(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *UserController) Disable() {
	if ctl.isForbidden(userResource, SaveAction) {
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
	var ob *models.User = &models.User{
		Id:         int64(_id),
		EnableFlag: false,
	}
	err = user.UpdateUserEnable(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}
