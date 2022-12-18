package api

import (
	"errors"
	"go-iot/models"
	user "go-iot/models/base"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	ns := web.NewNamespace("/api",
		web.NSRouter("/login", &LoginController{}, "post:LoginJson"),
		web.NSRouter("/logout", &LogoutController{}, "post:Logout"),
	)
	web.AddNamespace(ns)
}

type LoginController struct {
	RespController
}

func (ctl *LoginController) LoginJson() {
	var ob = struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{}

	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	u, err := user.GetUserByEntity(models.User{Username: ob.Username})
	if err != nil {
		ctl.RespError(err)
		return
	}
	if u == nil {
		ctl.RespError(errors.New("username or password invalid"))
		return
	}
	u1 := models.User{
		Username: ob.Username,
		Password: ob.Password,
	}
	user.Md5Pwd(&u1)
	old, err := user.GetUser(u.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if u1.Password != old.Password {
		ctl.RespError(errors.New("username or password invalid"))
		return
	}

	permission, err := user.GetPermissionByUserId(u.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var actionMap = map[string]bool{}
	for _, p := range permission.Permissions {
		for _, ac := range p.ActionEntitySet {
			actionMap[ac.Action] = true
		}
	}
	session := defaultSessionManager.Login(&ctl.Controller, u)
	session.SetPermission(actionMap)
	ctl.RespOk()
}

type LogoutController struct {
	AuthController
}

func (ctl *LogoutController) Logout() {
	defaultSessionManager.Logout(&ctl.AuthController)
}
