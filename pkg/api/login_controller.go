package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	user "go-iot/pkg/models/base"
)

func init() {
	web.RegisterAPI("/login", "POST", &LoginController{}, "LoginJson")
	web.RegisterAPI("/logout", "POST", &LogoutController{}, "Logout")
}

type LoginController struct {
	web.RespController
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

	err = ctl.login(ob.Username, ob.Password, ob.Expires)
	if err != nil {
		ctl.RespError(err)
		return
	}

	ctl.RespOkData(ctl.GetSession().Sessionid)
}

func (c *LoginController) login(username, password string, expire int) error {
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
	session := c.NewSession(expire)
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
