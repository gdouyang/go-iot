package notify

import (
	"errors"
	"go-iot/pkg/models"

	"go-iot/pkg/core/es/orm"
)

// 分页查询设备
func PageNotify(page *models.PageQuery, createId *int64) (*models.PageResult[models.Notify], error) {
	var pr *models.PageResult[models.Notify]
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Notify{})
	qs = qs.FilterTerm(page.Condition...)
	if createId != nil {
		qs = qs.Filter("createId", *createId)
	}
	qs.SearchAfter = page.SearchAfter
	var result []models.Notify
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

func ListAll(ob *models.Notify, createId *int64) ([]models.Notify, error) {
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Notify{})

	if len(ob.Name) > 0 {
		qs = qs.Filter("name__contains", ob.Name)
	}
	if len(ob.Type) > 0 {
		qs = qs.Filter("type", ob.Type)
	}

	if len(ob.State) > 0 {
		qs = qs.Filter("State", ob.State)
	}
	if createId != nil {
		qs = qs.Filter("createId", *createId)
	}

	var result []models.Notify
	_, err := qs.All(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func AddNotify(ob *models.Notify) error {
	//插入数据
	o := orm.NewOrm()
	ob.State = models.Stopped
	ob.CreateTime = models.NewDateTime()
	_, err := o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateNotify(ob *models.Notify) error {
	o := orm.NewOrm()
	_, err := o.Update(ob, "Name", "Config", "Template", "Type", "Desc")
	if err != nil {
		return err
	}
	return nil
}

func UpdateNotifyState(ob *models.Notify) error {
	if ob.Id == 0 {
		return errors.New("id must be present")
	}
	o := orm.NewOrm()
	_, err := o.Update(ob, "state")
	if err != nil {
		return err
	}
	return nil
}

func DeleteNotify(ob *models.Notify) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		return err
	}
	return nil
}

func GetNotify(id int64) (*models.Notify, error) {
	o := orm.NewOrm()

	p := models.Notify{Id: id}
	err := o.Read(&p, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}

func GetNotifyMust(id int64) (*models.Notify, error) {
	p, err := GetNotify(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("notify not exist")
	}
	return p, nil
}
