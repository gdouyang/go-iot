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
		web.NSRouter("/:id", &RoleController{}, "get:Get"),
		web.NSRouter("/:id", &RoleController{}, "delete:Delete"),
		web.NSRouter("/ref-menus/:id", &RoleController{}, "get:RefMenus"),
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
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (ctl *RoleController) Get() {
	if ctl.isForbidden(roleResource, QueryAction) {
		return
	}
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	u, err := role.GetRole(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(u)
}

func (ctl *RoleController) Add() {
	if ctl.isForbidden(roleResource, CretaeAction) {
		return
	}
	var ob role.RoleDTO
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = role.AddRole(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *RoleController) Update() {
	if ctl.isForbidden(roleResource, SaveAction) {
		return
	}
	var ob role.RoleDTO
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = role.UpdateRole(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *RoleController) Delete() {
	if ctl.isForbidden(roleResource, DeleteAction) {
		return
	}
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var ob *models.Role = &models.Role{
		Id: int64(_id),
	}
	err = role.DeleteRole(ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *RoleController) RefMenus() {
	if ctl.isForbidden(roleResource, QueryAction) {
		return
	}
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	list, err := role.GetAuthResourctByRole(int64(_id), role.ResTypeRole)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var result []struct {
		models.AuthResource
		Action []role.MenuAction `json:"action"`
	}
	for _, r := range list {
		var ac []role.MenuAction
		err := json.Unmarshal([]byte(r.Action), &ac)
		if err != nil {
			ctl.RespError(err)
			return
		}
		var r1 = struct {
			models.AuthResource
			Action []role.MenuAction `json:"action"`
		}{
			AuthResource: r,
			Action:       ac,
		}
		result = append(result, r1)
	}
	ctl.RespOkData(result)
}
