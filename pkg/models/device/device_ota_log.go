package models

import (
	"go-iot/pkg/es/orm"
	"go-iot/pkg/models"
)

func AddOtaLog(ob *models.DeviceOtaLog) error {
	o := orm.NewOrm()
	ob.CreateTime = models.NewDateTime()
	ob.UpdateTime = models.NewDateTime()
	_, err := o.Insert(ob)
	return err
}

func UpdateOtaLog(ob *models.DeviceOtaLog) error {
	o := orm.NewOrm()
	ob.UpdateTime = models.NewDateTime()
	_, err := o.Update(ob, "CurrentChunk", "Status", "Message", "UpdateTime")
	return err
}

// 分页查询OTA日志
func PageOtaLog(page *models.PageQuery) (*models.PageResult[models.DeviceOtaLog], error) {
	var pr *models.PageResult[models.DeviceOtaLog]
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.DeviceOtaLog{})
	qs = qs.FilterTerm(page.Condition...)
	qs.SearchAfter = page.SearchAfter

	var result []models.DeviceOtaLog
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
func DeleteOtaLog(id int64) error {
	o := orm.NewOrm()
	_, err := o.Delete(&models.DeviceOtaLog{Id: id})
	if err != nil {
		return err
	}
	return err
}
