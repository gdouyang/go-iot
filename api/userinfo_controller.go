package api

import (
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
	permission, err := user.GetPermissionByUserId(u1.Id)
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
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	ob.Id = ctl.GetCurrentUser().Id
	err = user.UpdateUser(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *UserInfoController) UpdatePwd() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob struct {
		Password    string `json:"password"`
		PasswordOld string `json:"passwordOld"`
	}
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if len(ob.PasswordOld) == 0 {
		resp = models.JsonRespError(errors.New("passwrodOld not be empty"))
		return
	}
	if len(ob.Password) == 0 {
		resp = models.JsonRespError(errors.New("passwrod not be empty"))
		return
	}
	u1 := models.User{
		Id:       ctl.GetCurrentUser().Id,
		Username: ctl.GetCurrentUser().Username,
		Password: ob.PasswordOld,
	}
	user.Md5Pwd(&u1)
	old, err := user.GetUser(u1.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if old.Password != u1.Password {
		resp = models.JsonRespError(errors.New("old password is incorrect"))
		return
	}
	//
	u := models.User{
		Id:       u1.Id,
		Username: u1.Username,
		Password: ob.Password,
	}
	err = user.UpdateUserPwd(&u)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}
