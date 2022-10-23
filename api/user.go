package api

import (
	"encoding/json"
	"go-iot/models"
	user "go-iot/models/base"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	ns := web.NewNamespace("/api/user",
		web.NSRouter("/list", &UserController{}, "post:List"),
		web.NSRouter("/", &UserController{}, "post:Add"),
		web.NSRouter("/", &UserController{}, "put:Update"),
		web.NSRouter("/:id", &UserController{}, "delete:Delete"),
	)
	web.AddNamespace(ns)
}

type UserController struct {
	AuthController
}

func (ctl *UserController) List() {
	var ob models.PageQuery
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)

	res, err := user.ListUser(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		ctl.Data["json"] = &res
	}
	ctl.ServeJSON()
}

func (ctl *UserController) Add() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.User
	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}

	err = user.AddUser(&ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}

func (ctl *UserController) Update() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.User
	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	err = user.UpdateUser(&ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}

func (ctl *UserController) Delete() {
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
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}
