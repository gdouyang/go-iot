package models

import (
	"go-iot/pkg/es/orm"
	"go-iot/pkg/models"
)

func AddOtaPackage(ob *models.DeviceOtaPackage) error {
	o := orm.NewOrm()
	ob.CreateTime = models.NewDateTime()
	ob.UpdateTime = models.NewDateTime()
	_, err := o.Insert(ob)
	return err
}

func UpdateOtaPackage(ob *models.DeviceOtaPackage) error {
	o := orm.NewOrm()
	ob.UpdateTime = models.NewDateTime()
	_, err := o.Update(ob, "FileName", "UpdateTime")
	return err
}

// 分页查询OTA包
func PageOtaPackage(page *models.PageQuery) (*models.PageResult[models.DeviceOtaPackage], error) {
	var pr *models.PageResult[models.DeviceOtaPackage]
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.DeviceOtaPackage{})
	qs = qs.FilterTerm(page.Condition...)
	qs.SearchAfter = page.SearchAfter

	var result []models.DeviceOtaPackage
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

// 删除OTA日志
func DeleteOtaPackage(id int64) error {
	o := orm.NewOrm()
	_, err := o.Delete(&models.DeviceOtaPackage{Id: id})
	if err != nil {
		return err
	}
	return err
}

func GetOta(otaId int) (*models.DeviceOtaPackage, error) {
	o := orm.NewOrm()
	p := models.DeviceOtaPackage{Id: int64(otaId)}
	err := o.Read(&p, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}
