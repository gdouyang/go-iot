package base

import (
	"encoding/json"
	"fmt"
	"go-iot/models"

	"github.com/beego/beego/v2/client/orm"
)

type RolePermissionDTO struct {
	Id          int64           `json:"id"`
	Name        string          `json:"name"`
	Permissions []PermissionDTO `json:"permissions"`
}

type PermissionDTO struct {
	RoleId          string                `json:"roleId"`
	PermissionId    string                `json:"permissionId"`
	PermissionName  string                `json:"permissionName"`
	Actions         string                `json:"actions"`
	ActionEntitySet []PermissionActionDTO `json:"actionEntitySet"`
}

type PermissionActionDTO struct {
	Action       string `json:"action"`
	Describe     string `json:"describe"`
	AefaultCheck bool   `json:"defaultCheck"`
}

type MenuAction struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func GetPermissionByRoleId(roleId int64, concatParentCode bool) (*RolePermissionDTO, error) {
	r := RolePermissionDTO{}
	if roleId == 0 {
		return &r, nil
	}
	role, err := GetRole(roleId)
	if err != nil {
		return nil, err
	}
	r.Name = role.Name
	refMenus, err := GetAuthResourctByRole(roleId, "ROLE")
	if err != nil {
		return nil, err
	}
	var list []models.MenuResource
	for _, ar := range refMenus {
		mr := models.MenuResource{}
		mr.Code = ar.Code
		mr.Action = ar.Action
		list = append(list, mr)
	}
	permissions := getPermissions(roleId, list, concatParentCode)
	r.Permissions = permissions
	return &r, nil
}

func GetAllMenuResource() ([]models.MenuResource, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(&models.MenuResource{})
	var result []models.MenuResource
	_, err := qs.All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetAuthResourctByRole(roleId int64, resType string) ([]models.AuthResource, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(&models.AuthResource{})
	qs = qs.Filter("objId", roleId).Filter("ResType", resType)
	var result []models.AuthResource
	_, err := qs.All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getPermissions(roleId int64, list []models.MenuResource, concatParentCode bool) []PermissionDTO {

	var result []PermissionDTO

	for _, m := range list {
		p := PermissionDTO{}
		p.RoleId = fmt.Sprintf("%d", roleId)
		p.PermissionId = m.Code
		p.PermissionName = m.Name

		var pactions []PermissionActionDTO
		if len(m.Action) > 0 {
			var actions []MenuAction
			json.Unmarshal([]byte(m.Action), &actions)
			for _, ma := range actions {
				a := PermissionActionDTO{}
				if concatParentCode {
					a.Action = fmt.Sprintf("%s:%s", m.Code, ma.Id)
				} else {
					a.Action = ma.Id
				}
				a.Describe = ma.Name
				pactions = append(pactions, a)
			}
			if len(actions) > 0 {
				p.ActionEntitySet = (pactions)
			}
		}
		result = append(result, p)
	}

	return result
}
