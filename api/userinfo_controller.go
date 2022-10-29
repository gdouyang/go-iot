package api

import (
	"encoding/json"
	"errors"
	"go-iot/models"
	user "go-iot/models/base"

	"github.com/beego/beego/v2/server/web"
)

// 产品管理
func init() {
	ns := web.NewNamespace("/api/user-info",
		web.NSRouter("/", &UserInfoController{}, "get:Get"),
		web.NSRouter("/save-basic", &UserInfoController{}, "put:SaveBasic"),
		web.NSRouter("/update-pwd", &UserInfoController{}, "put:UpdatePwd"),
	)
	web.AddNamespace(ns)
}

type UserInfoController struct {
	AuthController
}

func (ctl *UserInfoController) Get() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	u := ctl.GetCurrentUser()
	if u == nil {
		resp = models.JsonRespError(errors.New("user not login"))
		return
	}
	u1 := *u
	u1.Password = ""
	roles, err := user.GetUserRelRoleByUserId(u1.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	roleId := int64(0)
	if len(roles) > 0 {
		roleId = roles[0].RoleId
	}
	permission, err := user.GetPermissionByRoleId(roleId, true)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
	resp.Data = struct {
		models.User
		Role *user.RolePermissionDTO `json:"role"`
	}{User: u1, Role: permission}
}

func (ctl *UserInfoController) SaveBasic() {
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

func (ctl *UserInfoController) UpdatePwd() {
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
	err = user.UpdateUserPwd(&ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}
