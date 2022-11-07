package models

import (
	"encoding/json"
	"errors"
	"time"

	"go-iot/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 分页查询设备
func ListDevice(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.Device
	err := json.Unmarshal(page.Condition, &dev)
	if err != nil {
		return nil, err
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Device{})

	id := dev.Id
	if len(id) > 0 {
		qs = qs.Filter("id", id)
	}
	if len(dev.Name) > 0 {
		qs = qs.Filter("name__contains", dev.Name)
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Device
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func AddDevice(ob *models.Device) error {
	if len(ob.Id) == 0 || len(ob.Name) == 0 {
		return errors.New("id, name not be empty")
	}
	rs, err := GetDevice(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("device is exist")
	}
	ob.State = models.NoActive
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = time.Now()
	_, err = o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDevice(ob *models.Device) error {
	if len(ob.Id) == 0 {
		return errors.New("id not be empty")
	}
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "Name")
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

// 更新在线状态
func UpdateOnlineStatus(state string, id string) error {
	if len(id) == 0 {
		return errors.New("id not be empty")
	}
	if len(state) == 0 {
		return errors.New("state not be empty")
	}
	var ob models.Device = models.Device{Id: id, State: state}
	o := orm.NewOrm()
	_, err := o.Update(ob, "State")
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteDevice(ob *models.Device) error {
	if len(ob.Id) == 0 {
		return errors.New("id not be empty")
	}
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		logs.Error("delete fail", err)
		return err
	}
	return nil
}

func GetDevice(deviceId string) (*models.Device, error) {
	if len(deviceId) == 0 {
		return nil, errors.New("deviceId not be empty")
	}
	o := orm.NewOrm()
	p := models.Device{Id: deviceId}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}
