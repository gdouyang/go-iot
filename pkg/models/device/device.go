package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go-iot/pkg/core"
	"go-iot/pkg/models"

	"go-iot/pkg/es/orm"
)

func DeviceIdValid(deviceId string) bool {
	matched, _ := regexp.Match("^[0-9a-zA-Z_\\-]+$", []byte(deviceId))
	return matched
}

// 分页查询设备
func PageDevice(page *models.PageQuery, createId *int64) (*models.PageResult[models.Device], error) {
	var pr *models.PageResult[models.Device]
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Device{})
	qs = qs.FilterTerm(page.Condition...)
	qs.SearchAfter = page.SearchAfter
	if createId != nil {
		qs = qs.Filter("CreateId", *createId)
	}

	var result []models.Device
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
	if len(ob.Id) > 32 {
		return errors.New("设备ID长度不能超过32")
	}
	if !DeviceIdValid(ob.Id) {
		return errors.New("设备ID格式错误")
	}
	ob.DeviceType = strings.TrimSpace(ob.DeviceType)
	if len(ob.DeviceType) == 0 {
		ob.DeviceType = core.DEVICE
	}
	if ob.DeviceType == core.SUBDEVICE {
		if len(ob.ParentId) == 0 {
			return errors.New("子设备需要指定parentId")
		}
		gw, err := GetDevice(ob.ParentId)
		if err != nil {
			return err
		}
		if gw == nil {
			return errors.New("网关不存在")
		}
		if gw.DeviceType != core.GATEWAY {
			return errors.New("父级设备不是网关")
		}
	} else {
		ob.ParentId = ""
	}
	rs, err := GetDevice(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("设备已存在")
	}
	ob.State = core.NoActive
	en := ob.ToEnitty()
	if len(en.DeviceType) == 0 {
		en.DeviceType = core.DEVICE
	}
	en.CreateTime = models.NewDateTime()
	//插入数据
	o := orm.NewOrm()
	_, err = o.Insert(&en)
	return err
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
	return err
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
	return nil
}

func DeleteDevice(deviceId string) error {
	if len(deviceId) == 0 {
		return errors.New("id must be present")
	}
	o := orm.NewOrm()
	_, err := o.Delete(&models.Device{Id: deviceId})
	return err
}

func GetDevice(deviceId string) (*models.DeviceModel, error) {
	if len(deviceId) == 0 {
		return nil, errors.New("deviceId must be present")
	}
	o := orm.NewOrm()
	p := models.Device{Id: deviceId}
	err := o.Read(&p, "id")
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

func GetDeviceAndCheckCreateId(deviceId string, createId int64) (*models.DeviceModel, error) {
	ob, err := GetDeviceMust(deviceId)
	if err != nil {
		return nil, err
	}
	if ob.CreateId != createId {
		return nil, errors.New("device is not you created")
	}
	return ob, nil
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
