package api

import (
	"encoding/json"
	"go-iot/models"
	user "go-iot/models/base"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

// 产品管理
func init() {
	ns := web.NewNamespace("/api/user",
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
	id := ctl.Ctx.Input.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		return
	}
	u, err := user.GetUser(int64(_id))
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		return
	}
	u.Password = ""
	resp = models.JsonRespOk()
	resp.Data = &u
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
