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
	qs := o.QueryTable("device")

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

func AddDevice(ob *models.Device) error {
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
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "Name")
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

// 根据SN与provider来更新设备在线状态
func UpdateOnlineStatus(onlineStatus string, id string) error {
	//更新数据
	db, _ := models.GetDb()
	defer db.Close()
	stmt, err := db.Prepare(`
	update device 
	set online_status_ = ?
	where id_ = ?
	`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(onlineStatus, id)
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteDevice(ob *models.Device) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		logs.Error("delete fail", err)
		return err
	}
	return nil
}

func GetDevice(deviceId string) (models.Device, error) {
	o := orm.NewOrm()
	p := models.Device{Id: deviceId}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return models.Device{}, err
	} else if err == orm.ErrMissPK {
		return models.Device{}, err
	} else {
		return p, nil
	}
}
