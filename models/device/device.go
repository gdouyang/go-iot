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
func ListDevice(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.Device
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Device{})

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
		Data:     result}

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
	if len(rs.Id) > 0 {
		return errors.New("设备已存在")
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
func UpdateOnlineStatus(onlineStatus string, id string) error {
	if len(id) == 0 {
		return errors.New("id not be empty")
	}
	if len(onlineStatus) == 0 {
		return errors.New("onlineStatus not be empty")
	}
	var ob models.Device = models.Device{Id: id, OnlineStatus: onlineStatus}
	o := orm.NewOrm()
	_, err := o.Update(ob, "OnlineStatus")
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

func GetDevice(deviceId string) (models.Device, error) {
	if len(deviceId) == 0 {
		return models.Device{}, errors.New("deviceId not be empty")
	}
	o := orm.NewOrm()
	p := models.Device{Id: deviceId}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return models.Device{}, nil
	} else if err == orm.ErrMissPK {
		return models.Device{}, err
	} else {
		return p, nil
	}
}
