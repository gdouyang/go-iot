package models

import (
	"encoding/json"
	"errors"
	"time"

	"go-iot/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func init() {
}

// 分页查询设备
func ListProduct(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.Product
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable("product")

	id := dev.Id
	if len(id) > 0 {
		qs.Filter("id", id)
	}
	if len(dev.Name) > 0 {
		qs.Filter("name__contains", dev.Name)
	}
	qs.Offset(page.PageOffset())
	qs.Limit(page.PageSize)

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Product
	_, err = qs.All(&result)
	if err != nil {
		return nil, err
	}

	pr = &models.PageResult{
		PageSize: page.PageSize,
		PageNum:  page.PageNum,
		Total:    count,
		List:     result}

	return pr, nil
}

func AddProduct(ob *models.Product) error {
	rs, err := GetProduct(ob.Id)
	if err != nil {
		return err
	}
	if len(rs.Id) > 0 {
		return errors.New("设备已存在!")
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

func UpdateProduct(ob *models.Product) error {
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "Name", "TypeId")
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteProduct(ob *models.Product) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		logs.Error("delete fail", err)
		return err
	}
	return nil
}

func GetProduct(id string) (models.Product, error) {

	o := orm.NewOrm()

	p := models.Product{Id: id}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return models.Product{}, err
	} else if err == orm.ErrMissPK {
		return models.Product{}, err
	} else {
		return p, nil
	}
}
