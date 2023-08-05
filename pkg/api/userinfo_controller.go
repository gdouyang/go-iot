package api

import (
	"errors"
	"go-iot/pkg/models"
	user "go-iot/pkg/models/base"

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
	u := ctl.GetCurrentUser()
	if u == nil {
		ctl.RespError(errors.New("user not login"))
		return
	}
	u1 := *u
	u1.Password = ""
	permission, err := user.GetPermissionByUserId(u1.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(struct {
		models.User
		Role *user.RolePermissionDTO `json:"role"`
	}{User: u1, Role: permission})
}

func (ctl *UserInfoController) SaveBasic() {
	var ob user.UserDTO
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.Id = ctl.GetCurrentUser().Id
	err = user.UpdateUserBaseInfo(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *UserInfoController) UpdatePwd() {
	var ob struct {
		Password    string `json:"password"`
		PasswordOld string `json:"passwordOld"`
	}
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if len(ob.PasswordOld) == 0 {
		ctl.RespError(errors.New("passwrodOld must be present"))
		return
	}
	if len(ob.Password) == 0 {
		ctl.RespError(errors.New("passwrod must be present"))
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
		ctl.RespError(err)
		return
	}
	if old.Password != u1.Password {
		ctl.RespError(errors.New("旧密码错误"))
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
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}
