package api

import (
	"go-iot/models"
	role "go-iot/models/base"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

var roleResource = Resource{
	Id:   "role-mgr",
	Name: "角色管理",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

// 产品管理
func init() {
	ns := web.NewNamespace("/api/role",
		web.NSRouter("/page", &RoleController{}, "post:List"),
		web.NSRouter("/", &RoleController{}, "post:Add"),
		web.NSRouter("/", &RoleController{}, "put:Update"),
		web.NSRouter("/:id", &RoleController{}, "get:Get"),
		web.NSRouter("/:id", &RoleController{}, "delete:Delete"),
	)
	web.AddNamespace(ns)

	regResource(roleResource)
}

type RoleController struct {
	AuthController
}

// 查询列表
func (ctl *RoleController) List() {
	if ctl.isForbidden(roleResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := role.ListRole(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {
		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
}

func (ctl *RoleController) Get() {
	if ctl.isForbidden(roleResource, QueryAction) {
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
	u, err := role.GetRole(int64(_id))
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOkData(u)
}

func (ctl *RoleController) Add() {
	if ctl.isForbidden(roleResource, CretaeAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Role
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = role.AddRole(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonResp{Success: true}
}

func (ctl *RoleController) Update() {
	if ctl.isForbidden(roleResource, SaveAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Role
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = role.UpdateRole(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonResp{Success: true}
}

func (ctl *RoleController) Delete() {
	if ctl.isForbidden(roleResource, DeleteAction) {
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
	var ob *models.Role = &models.Role{
		Id: int64(_id),
	}
	err = role.DeleteRole(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonResp{Success: true}
}
