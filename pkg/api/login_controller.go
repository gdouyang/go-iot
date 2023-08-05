package api

import (
	"errors"
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
		Expires  int    `json:"expires"`
	}{}

	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	err = ctl.login(&ctl.RespController, ob.Username, ob.Password, ob.Expires)
	if err != nil {
		ctl.RespError(err)
		return
	}

	ctl.RespOkData(ctl.GetSession().Sessionid)
}

func (c *LoginController) login(ctl *RespController, username, password string, expire int) error {
	u, err := user.GetUserByEntity(models.User{Username: username})
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("账号或密码错误")
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
		return errors.New("账号或密码错误")
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
	session := ctl.NewSession(expire)
	session.SetAttribute("user", u)
	session.SetPermission(actionMap)
	return nil
}

type LogoutController struct {
	AuthController
}

func (ctl *LogoutController) Logout() {
	ctl.AuthController.Logout(&ctl.AuthController)
}
