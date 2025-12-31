package models

import (
	"go-iot/pkg/es/orm"
	"go-iot/pkg/models"
)

func AddOtaFile(ob *models.OtaFile) error {
	o := orm.NewOrm()
	ob.CreateTime = models.NewDateTime()
	_, err := o.Insert(ob)
	return err
}

func PageOtaFile(page *models.PageQuery) (*models.PageResult[models.OtaFile], error) {
	var pr *models.PageResult[models.OtaFile]
	o := orm.NewOrm()
	qs := o.QueryTable(models.OtaFile{})
	qs = qs.FilterTerm(page.Condition...)
	qs.SearchAfter = page.SearchAfter

	var result []models.OtaFile
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

func DeleteOtaFile(id int64) error {
	o := orm.NewOrm()
	_, err := o.Delete(&models.OtaFile{Id: id})
	return err
}

func UpdateOtaFile(ob *models.OtaFile) error {
	o := orm.NewOrm()
	_, err := o.Update(ob, "Name", "Path", "Size", "ProductId")
	return err
}

func GetOtaFile(id int64) (*models.OtaFile, error) {
	o := orm.NewOrm()
	var file models.OtaFile
	file.Id = id
	err := o.Read(&file, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &file, nil
}
