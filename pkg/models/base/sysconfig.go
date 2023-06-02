package base

import (
	"errors"
	"go-iot/pkg/models"

	"go-iot/pkg/core/es/orm"
)

func AddSysconfig(ob *models.SystemConfig) error {
	rs, err := GetSysconfig(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("id exist")
	}
	//插入数据
	o := orm.NewOrm()
	_, err = o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateSysconfig(ob *models.SystemConfig) error {
	o := orm.NewOrm()
	_, err := o.Update(ob, "config")
	if err != nil {
		return err
	}
	return nil
}

func GetSysconfig(id string) (*models.SystemConfig, error) {

	o := orm.NewOrm()

	p := models.SystemConfig{Id: id}
	err := o.Read(&p, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}
