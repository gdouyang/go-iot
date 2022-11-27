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
	web.Controller
}

func (ctl *LoginController) LoginJson() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob = struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{}

	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}

	u, err := user.GetUserByEntity(models.User{Username: ob.Username})
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if u == nil {
		resp = models.JsonRespError(errors.New("username or password invalid"))
		return
	}
	u1 := models.User{
		Username: ob.Username,
		Password: ob.Password,
	}
	user.Md5Pwd(&u1)
	old, err := user.GetUser(u.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if u1.Password != old.Password {
		resp = models.JsonRespError(errors.New("username or password invalid"))
		return
	}

	permission, err := user.GetPermissionByUserId(u.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	var actionMap = map[string]bool{}
	for _, p := range permission.Permissions {
		for _, ac := range p.ActionEntitySet {
			actionMap[ac.Action] = true
		}
	}
	resp = models.JsonRespOk()
	session := defaultSessionManager.Login(&ctl.Controller, u)
	session.SetPermission(actionMap)
}

type LogoutController struct {
	AuthController
}

func (ctl *LogoutController) Logout() {
	defaultSessionManager.Logout(&ctl.AuthController)
}
