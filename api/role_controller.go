package api

import (
	"encoding/json"
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
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)

	res, err := role.ListRole(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {
		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
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
	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	err = role.AddRole(&ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
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
	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	err = role.UpdateRole(&ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
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
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}
