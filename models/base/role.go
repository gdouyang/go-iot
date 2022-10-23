package base

import (
	"encoding/json"
	"errors"
	"go-iot/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 分页查询设备
func ListRole(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var n models.Role
	err1 := json.Unmarshal(page.Condition, &n)
	if err1 != nil {
		return nil, err1
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Role{})

	id := n.Id
	if id > 0 {
		qs = qs.Filter("id", id)
	}
	if len(n.Name) > 0 {
		qs = qs.Filter("name__contains", n.Name)
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Role
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func AddRole(ob *models.Role) error {
	rs, err := GetRoleByEntity(models.Role{Name: ob.Name})
	if err != nil {
		return err
	}
	if rs.Id > 0 {
		return errors.New("名称已存在")
	}
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = time.Now()
	_, err = o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateRole(ob *models.Role) error {
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "Name")
	if err != nil {
		return err
	}
	return nil
}

func DeleteRole(ob *models.Role) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		logs.Error("delete fail", err)
		return err
	}
	return nil
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
