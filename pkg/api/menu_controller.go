package api

import (
	"go-iot/pkg/api/web"
	user "go-iot/pkg/models/base"
	"net/http"
)

func init() {
	web.RegisterAPI("/menu/list", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
}
