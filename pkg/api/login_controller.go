package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	user "go-iot/pkg/models/base"
	"net/http"

	logs "go-iot/pkg/logger"
)

func init() {
	// 登录
	web.RegisterAPI("/login", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := web.NewController(w, r)
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

		err = login(ctl, ob.Username, ob.Password, ob.Expires)
		if err != nil {
			ctl.RespError(err)
			return
		}

		ctl.RespOkData(ctl.GetSession().Sessionid)
	})
	// 登出
	web.RegisterAPI("/logout", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		ctl.Logout()
	})
}

func login(c *web.RespController, username, password string, expire int) error {
	u, err := user.GetUserByEntity(models.User{Username: username})
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("账号或密码错误")
	}
	old, err := user.GetUser(u.Id)
	if err != nil {
		return err
	}
	if old == nil {
		return errors.New("账号或密码错误")
	}
	matched, needUpgrade := user.CheckPassword(old.Password, username, password)
	if !matched {
		return errors.New("账号或密码错误")
	}
	// 旧 MD5 哈希透明升级为 bcrypt
	if needUpgrade {
		if err := user.UpdateUserPwd(&models.User{
			Id:       old.Id,
			Username: username,
			Password: password,
		}); err != nil {
			logs.Errorf("password lazy upgrade failed for user %s: %v", username, err)
		}
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
