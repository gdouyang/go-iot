package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/core/es"
	"go-iot/pkg/models"

	"github.com/beego/beego/v2/client/orm"
)

func DeviceIdValid(deviceId string) bool {
	matched, _ := regexp.Match("^[0-9a-zA-Z_\\-]+$", []byte(deviceId))
	return matched
}

// 分页查询设备
func PageDevice(page *models.PageQuery[models.Device], createId int64) (*models.PageResult[models.Device], error) {
	var pr *models.PageResult[models.Device]
	var dev models.Device = page.Condition
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
	if len(dev.State) > 0 {
		qs = qs.Filter("State", dev.State)
	}

	if len(dev.ProductId) > 0 {
		qs = qs.Filter("ProductId", dev.ProductId)
	}
	if len(dev.ParentId) > 0 {
		qs = qs.Filter("parentId", dev.ParentId)
	}
	if len(dev.DeviceType) > 0 {
		qs = qs.Filter("deviceType", dev.DeviceType)
	}
	qs = qs.Filter("CreateId", createId)

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Device
	_, err = qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime").All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}
func PageDeviceEs(page *models.PageQuery[models.DeviceModel], createId int64) (*models.PageResult[models.DeviceModel], error) {
	var dev models.DeviceModel = page.Condition
	param := map[string]interface{}{}
	id := dev.Id
	if len(id) > 0 {
		param["id"] = id
	}
	if len(dev.Name) > 0 {
		param["name$LIKE"] = dev.Name
	}
	if len(dev.State) > 0 {
		param["state"] = dev.State
	}
	if len(dev.ProductId) > 0 {
		param["productId"] = dev.ProductId
	}
	if len(dev.ParentId) > 0 {
		param["parentId"] = dev.ParentId
	}
	if len(dev.DeviceType) > 0 {
		param["deviceType"] = dev.DeviceType
	}
	if len(dev.Tag) > 0 {
		for key, value := range dev.Tag {
			param["tag."+key] = value
		}
	}
	total, result, err := es.PageDevice[models.DeviceModel](page.PageOffset(), page.PageSize, param)
	if err != nil {
		return nil, err
	}
	var pr *models.PageResult[models.DeviceModel]
	p := models.PageUtil(total, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func ListClientDeviceByProductId(productId string) ([]string, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(models.Device{})
	qs = qs.Filter("ProductId", productId).Filter("State", core.OFFLINE)

	var result []models.Device
	_, err := qs.All(&result, "id")
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, v := range result {
		ids = append(ids, v.Id)
	}
	return ids, nil
}

func AddDevice(ob *models.DeviceModel) error {
	if len(ob.Id) == 0 || len(ob.Name) == 0 {
		return errors.New("id, name must be present")
	}
	if !DeviceIdValid(ob.Id) {
		return errors.New("deviceId is invalid")
	}
	if ob.DeviceType == "subdevice" && len(ob.ParentId) == 0 {
		return errors.New("subdevice must have gateway")
	}
	rs, err := GetDevice(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("device is exist")
	}
	ob.State = core.NoActive
	en := ob.ToEnitty()
	if len(en.DeviceType) == 0 {
		en.DeviceType = "device"
	}
	en.CreateTime = time.Now()
	//插入数据
	o := orm.NewOrm()
	_, err = o.Insert(&en)
	if err != nil {
		return err
	}
	b, err := json.Marshal(ob)
	if err != nil {
		return err
	}
	data := map[string]interface{}{}
	json.Unmarshal(b, &data)
	data["createTime"] = time.Now().Format("2006-01-02 15:04:05")
	err = es.AddDevice(ob.Id, data)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDevice(ob *models.Device) error {
	if len(ob.Id) == 0 {
		return errors.New("id must be present")
	}
	//更新数据
	o := orm.NewOrm()
	var columns []string
	var data map[string]interface{} = make(map[string]interface{})
	if len(ob.Name) > 0 {
		columns = append(columns, "Name")
		data["name"] = ob.Name
	}
	if len(ob.Desc) > 0 {
		columns = append(columns, "Desc")
		data["desc"] = ob.Desc
	}
	if len(ob.Metaconfig) > 0 {
		columns = append(columns, "Metaconfig")
		data["metaconfig"] = ob.Metaconfig
	}
	if len(columns) == 0 {
		return errors.New("no data to update")
	}
	_, err := o.Update(ob, columns...)
	if err != nil {
		return err
	}
	err = es.UpdateDevice(ob.Id, data)
	if err != nil {
		return err
	}
	return nil
}

// 更新在线状态
func UpdateOnlineStatus(id string, state string) error {
	if len(id) == 0 {
		return errors.New("id must be present")
	}
	err := UpdateOnlineStatusList([]string{id}, state)
	if err != nil {
		return err
	}
	return nil
}

func UpdateOnlineStatusList(ids []string, state string) error {
	if len(ids) == 0 {
		return errors.New("ids must be present")
	}
	if len(state) == 0 {
		return errors.New("state must be present")
	}
	o := orm.NewOrm()
	_, err := o.QueryTable(models.Device{}).Filter("id__in", ids).Update(orm.Params{"state": state})
	if err != nil {
		return err
	}
	err = es.UpdateOnlineStatusList(ids, state)
	if err != nil {
		return err
	}
	return nil
}

func DeleteDevice(deviceId string) error {
	if len(deviceId) == 0 {
		return errors.New("id must be present")
	}
	o := orm.NewOrm()
	_, err := o.Delete(&models.Device{Id: deviceId})
	if err != nil {
		return err
	}
	err = es.DeleteDevice(deviceId)
	if err != nil {
		return err
	}
	return nil
}

func GetDevice(deviceId string) (*models.DeviceModel, error) {
	if len(deviceId) == 0 {
		return nil, errors.New("deviceId must be present")
	}
	o := orm.NewOrm()
	p := models.Device{Id: deviceId}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		m := &models.DeviceModel{}
		m.FromEnitty(p)
		return m, nil
	}
}

func GetDeviceMust(deviceId string) (*models.DeviceModel, error) {
	p, err := GetDevice(deviceId)
	if err != nil {
		return nil, err
	} else if p == nil {
		return nil, fmt.Errorf("device [%s] not exist", deviceId)
	}
	return p, nil
}

func CountDeviceByProductId(productId string) (int64, error) {
	if len(productId) == 0 {
		return -1, errors.New("productId must be present")
	}
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Device{}).Filter("productId", productId)
	count, err := qs.Count()
	if err != nil {
		return -1, err
	}
	return count, nil
}
