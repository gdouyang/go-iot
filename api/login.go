package api

import (
	"encoding/json"
	"errors"
	"go-iot/models"
	user "go-iot/models/base"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	ns := web.NewNamespace("/api",
		web.NSRouter("/login", &LoginController{}, "post:LoginJson"),
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

	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
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
	resp = models.JsonResp{Success: true}
	session := defaultSessionManager.NewSession()
	session.Put("user", &u)
	ctl.Ctx.Output.Cookie("gsessionid", session.sessionid)
}
