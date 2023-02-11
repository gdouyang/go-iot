package api

import (
	user "go-iot/pkg/models/base"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	ns := web.NewNamespace("/api/menu",
		web.NSRouter("/list", &MenuController{}, "get:List"),
	)
	web.AddNamespace(ns)

	regResource(Resource{
		Id:   "menu-mgr",
		Name: "菜单资源",
		Action: []ResourceAction{
			QueryAction,
		},
	})
}

type MenuController struct {
	AuthController
}

func (ctl *MenuController) List() {
	u := ctl.GetCurrentUser()
	roles, err := user.GetUserRelRoleByUserId(u.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	roleId := int64(0)
	if len(roles) > 0 {
		roleId = roles[0].RoleId
	}
	permission, err := user.GetPermissionByRoleId(roleId, false)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(permission.Permissions)
}
