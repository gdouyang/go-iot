package base

import (
	"encoding/json"
	"errors"
	"go-iot/pkg/models"

	"go-iot/pkg/core/es/orm"
)

type RelType string

const (
	ResTypeRole RelType = "ROLE"
	ResTypeUser RelType = "USER"
)

// 分页查询设备
func PageRole(page *models.PageQuery, createId int64) (*models.PageResult[models.Role], error) {
	var pr *models.PageResult[models.Role]

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Role{})

	qs = qs.FilterTerm(page.Condition...)
	qs.SearchAfter = page.SearchAfter
	qs = qs.Filter("createId", createId)

	var result []models.Role
	_, err := qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime", "-id").All(&result)
	if err != nil {
		return nil, err
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	p.SearchAfter = qs.LastSort
	pr = &p

	return pr, nil
}

type RoleDTO struct {
	models.Role
	RuleRefMenus []struct {
		Code   string       `json:"code"`
		Action []MenuAction `json:"action"`
	} `json:"ruleRefMenus"`
}

func AddRole(ob *RoleDTO) error {
	rs, err := GetRoleByEntity(models.Role{Name: ob.Name})
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("名称已存在")
	}
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = models.NewDateTime()
	_, err = o.Insert(&ob.Role)
	if err != nil {
		return err
	}
	var auths []models.AuthResource
	for _, m := range ob.RuleRefMenus {
		ar := models.AuthResource{}
		ac, err := json.Marshal(m.Action)
		if err != nil {
			return err
		}
		ar.Code = m.Code
		ar.ObjId = ob.Id
		ar.ResType = string(ResTypeRole)
		ar.Action = string(ac)
		auths = append(auths, ar)
	}
	for _, ar := range auths {
		ar.ObjId = ob.Role.Id
		err = AddAuthResource(&ar)
		if err != nil {
			DeleteRole(&ob.Role)
			return err
		}
	}
	return nil
}

func UpdateRole(ob *RoleDTO) error {
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(&ob.Role, "Desc")
	if err != nil {
		return err
	}
	var auths []models.AuthResource
	for _, m := range ob.RuleRefMenus {
		ar := models.AuthResource{}
		ac, err := json.Marshal(m.Action)
		if err != nil {
			return err
		}
		ar.Code = m.Code
		ar.ObjId = ob.Id
		ar.ResType = string(ResTypeRole)
		ar.Action = string(ac)
		auths = append(auths, ar)
	}
	err = deleteRoleAuth(&ob.Role)
	if err != nil {
		return err
	}
	for _, ar := range auths {
		ar.ObjId = ob.Role.Id
		err = AddAuthResource(&ar)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteRole(ob *models.Role) error {
	err := deleteRoleAuth(ob)
	if err != nil {
		return err
	}
	o := orm.NewOrm()
	_, err = o.Delete(ob)
	if err != nil {
		return err
	}
	return err
}

func deleteRoleAuth(ob *models.Role) error {
	o := orm.NewOrm()
	_, err := o.Delete(&models.AuthResource{
		ResType: string(ResTypeRole),
		ObjId:   ob.Id,
	}, "ResType", "ObjId")
	return err
}

func GetRole(id int64) (*models.Role, error) {

	o := orm.NewOrm()

	p := models.Role{Id: id}
	err := o.Read(&p, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}

func GetRoleByEntity(p models.Role) (*models.Role, error) {

	o := orm.NewOrm()
	cols := []string{}
	if p.Id != 0 {
		cols = append(cols, "id")
	}
	if len(p.Name) > 0 {
		cols = append(cols, "name")
	}
	err := o.Read(&p, cols...)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}

func AddAuthResource(ob *models.AuthResource) error {
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = models.NewDateTime()
	_, err := o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}
