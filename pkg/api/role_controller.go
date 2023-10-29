package api

import (
	"encoding/json"
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	role "go-iot/pkg/models/base"
	"net/http"
	"strconv"
)

var roleResource = Resource{
	Id:   "role-mgr",
	Name: "角色管理",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

// 产品管理
func init() {
	getRoleAndCheckCreateId := func(ctl *AuthController, roleId int64) (*models.Role, error) {
		ob, err := role.GetRole(roleId)
		if err != nil {
			return nil, err
		}
		if ob.CreateId != ctl.GetCurrentUser().Id {
			return nil, errors.New("data is not you created")
		}
		return ob, nil
	}
	web.RegisterAPI("/role/page", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(roleResource, QueryAction) {
			return
		}
		var ob models.PageQuery
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}

		res, err := role.PageRole(&ob, ctl.GetCurrentUser().Id)
		if err != nil {
			ctl.RespError(err)
		} else {
			ctl.RespOkData(res)
		}
	})
	// 新增角色
	web.RegisterAPI("/role", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(roleResource, CretaeAction) {
			return
		}
		var ob role.RoleDTO
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ob.CreateId = ctl.GetCurrentUser().Id
		err = role.AddRole(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 修改角色
	web.RegisterAPI("/role", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(roleResource, SaveAction) {
			return
		}
		var ob role.RoleDTO
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getRoleAndCheckCreateId(ctl, ob.Id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		err = role.UpdateRole(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	web.RegisterAPI("/role/{id}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(roleResource, QueryAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getRoleAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		u, err := role.GetRole(int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOkData(u)
	})
	// 删除角色
	web.RegisterAPI("/role/{id}", "DELETE", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(roleResource, DeleteAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		var ob *models.Role = &models.Role{
			Id: int64(_id),
		}
		_, err = getRoleAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		err = role.DeleteRole(ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 查看角色的菜单
	web.RegisterAPI("/role/ref-menus/{id}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(roleResource, QueryAction) {
			return
		}
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		_, err = getRoleAndCheckCreateId(ctl, int64(_id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		list, err := role.GetAuthResourctByRole(int64(_id), role.ResTypeRole)
		if err != nil {
			ctl.RespError(err)
			return
		}
		var result []struct {
			models.AuthResource
			Action []role.MenuAction `json:"action"`
		}
		for _, r := range list {
			var ac []role.MenuAction
			err := json.Unmarshal([]byte(r.Action), &ac)
			if err != nil {
				ctl.RespError(err)
				return
			}
			var r1 = struct {
				models.AuthResource
				Action []role.MenuAction `json:"action"`
			}{
				AuthResource: r,
				Action:       ac,
			}
			result = append(result, r1)
		}
		ctl.RespOkData(result)
	})

	RegResource(roleResource)
}
