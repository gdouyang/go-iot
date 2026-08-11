package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	user "go-iot/pkg/models/base"
	"net/http"
)

// 用户信息
func init() {
	// 用户个人信息
	web.RegisterAPI("/user-info", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	// 修改基本信息
	web.RegisterAPI("/user-info/save-basic", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	// 修改本人密码
	web.RegisterAPI("/user-info/update-pwd", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
			ctl.RespErrorParam("passwrodOld")
			return
		}
		if len(ob.Password) == 0 {
			ctl.RespErrorParam("passwrod")
			return
		}
		cur := ctl.GetCurrentUser()
		old, err := user.GetUser(cur.Id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if old == nil {
			ctl.RespError(errors.New("user not found"))
			return
		}
		matched, _ := user.CheckPassword(old.Password, cur.Username, ob.PasswordOld)
		if !matched {
			ctl.RespError(errors.New("旧密码错误"))
			return
		}
		u := models.User{
			Id:       cur.Id,
			Username: cur.Username,
			Password: ob.Password,
		}
		err = user.UpdateUserPwd(&u)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
}
