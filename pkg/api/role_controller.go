package api

import (
	"encoding/json"
	"errors"
	"go-iot/pkg/models"
	role "go-iot/pkg/models/base"
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
		web.NSRouter("/page", &RoleController{}, "post:Page"),
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
func (ctl *RoleController) Page() {
	if ctl.isForbidden(roleResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := role.PageRole(&ob, ctl.GetCurrentUser().Id)
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
	_, err = ctl.getRoleAndCheckCreateId(int64(_id))
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
	ob.CreateId = ctl.GetCurrentUser().Id
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
	_, err = ctl.getRoleAndCheckCreateId(ob.Id)
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
	_, err = ctl.getRoleAndCheckCreateId(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
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
	_, err = ctl.getRoleAndCheckCreateId(int64(_id))
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

func (ctl *RoleController) getRoleAndCheckCreateId(roleId int64) (*models.Role, error) {
	ob, err := role.GetRole(roleId)
	if err != nil {
		return nil, err
	}
	if ob.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New("data is not you created")
	}
	return ob, nil
}
