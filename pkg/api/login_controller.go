package api

import (
	"errors"
	"go-iot/pkg/api/session"
	"go-iot/pkg/models"
	user "go-iot/pkg/models/base"

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

	err = ctl.login(&ctl.Controller, ob.Username, ob.Password)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (c *LoginController) login(ctl *web.Controller, username, password string) error {
	u, err := user.GetUserByEntity(models.User{Username: username})
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("username or password invalid")
	}
	u1 := models.User{
		Username: username,
		Password: password,
	}
	user.Md5Pwd(&u1)
	old, err := user.GetUser(u.Id)
	if err != nil {
		return err
	}
	if u1.Password != old.Password {
		return errors.New("username or password invalid")
	}

	permission, err := user.GetPermissionByUserId(u.Id)
	if err != nil {
		return err
	}
	var actionMap = map[string]bool{}
	for _, p := range permission.Permissions {
		for _, ac := range p.ActionEntitySet {
			actionMap[ac.Action] = true
		}
	}
	session := session.NewSession()
	session.Put("user", u)
	ctl.Ctx.Request.Header.Add("gsessionid", session.Sessionid)
	ctl.Ctx.Output.Cookie("gsessionid", session.Sessionid, int64(3600*24))
	session.SetPermission(actionMap)
	return nil
}

type LogoutController struct {
	AuthController
}

func (ctl *LogoutController) Logout() {
	ctl.AuthController.Logout(&ctl.AuthController)
}
