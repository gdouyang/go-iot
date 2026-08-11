package base

import (
	"encoding/json"
	"fmt"
	"go-iot/pkg/models"
	"sort"

	"go-iot/pkg/es/orm"
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
	// Sort 与 MenuResource.sort 一致，前端/角色树可按此升序展示
	Sort            int32                 `json:"sort"`
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

func GetPermissionByUserId(userId int64) (*RolePermissionDTO, error) {
	roles, err := GetUserRelRoleByUserId(userId)
	if err != nil {
		return nil, err
	}
	roleId := int64(0)
	if len(roles) > 0 {
		roleId = roles[0].RoleId
	}
	permission, err := GetPermissionByRoleId(roleId, true)
	if err != nil {
		return nil, err
	}
	return permission, nil
}

func GetPermissionByRoleId(roleId int64, concatParentCode bool) (*RolePermissionDTO, error) {
	r := RolePermissionDTO{}
	if roleId == 0 {
		menus, err := GetAllMenuResource()
		if err != nil {
			return nil, err
		}
		permissions := getPermissions(roleId, menus, concatParentCode)
		r.Permissions = permissions
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
		mr := models.MenuResource{
			Code:   ar.Code,
			Action: ar.Action,
			Sort:   ar.Sort,
		}
		// 补全名称与排序：授权表可能无 Sort，以菜单资源为准
		if full, err := GetMenuResourceByCode(ar.Code); err == nil && full != nil {
			if full.Name != "" {
				mr.Name = full.Name
			}
			mr.Sort = full.Sort
		}
		list = append(list, mr)
	}
	sortMenuResources(list)
	permissions := getPermissions(roleId, list, concatParentCode)
	r.Permissions = permissions
	return &r, nil
}

func GetAllMenuResource() ([]models.MenuResource, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(&models.MenuResource{})
	// 按 sort 升序；同 sort 时按 id 稳定（+ 前缀才会 FirstLower 到 ES 字段名）
	var result []models.MenuResource
	_, err := qs.OrderBy("+Sort", "+id").All(&result)
	if err != nil {
		return nil, err
	}
	// ES 排序字段映射若异常，内存再保底一次
	sortMenuResources(result)
	return result, nil
}

func sortMenuResources(list []models.MenuResource) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Sort != list[j].Sort {
			return list[i].Sort < list[j].Sort
		}
		return list[i].Id < list[j].Id
	})
}

func GetAuthResourctByRole(roleId int64, resType RelType) ([]models.AuthResource, error) {
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
	// 调用方应已排序；此处再保底，保证 /menu/list 与角色权限顺序稳定
	sortMenuResources(list)

	var result []PermissionDTO

	for _, m := range list {
		p := PermissionDTO{}
		p.RoleId = fmt.Sprintf("%d", roleId)
		p.PermissionId = m.Code
		p.PermissionName = m.Name
		p.Sort = m.Sort

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

func GetMenuResourceByCode(code string) (*models.MenuResource, error) {

	o := orm.NewOrm()

	p := models.MenuResource{Code: code}
	err := o.Read(&p, "Code")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}

func AddMenuResource(ob *models.MenuResource) error {
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = models.NewDateTime()
	_, err := o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateMenuResource(ob *models.MenuResource) error {
	// 同步注册表中的名称、动作、排序（启动时 initMenuResources 会刷新）
	o := orm.NewOrm()
	_, err := o.Update(ob, "Name", "Action", "Sort")
	if err != nil {
		return err
	}
	return nil
}
