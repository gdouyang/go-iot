package scene

import (
	"encoding/json"
	"errors"
	"go-iot/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// 分页查询
func ListAlarm(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.Alarm
	err := json.Unmarshal(page.Condition, &dev)
	if err != nil {
		return nil, err
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Alarm{})

	if len(dev.Name) > 0 {
		qs = qs.Filter("name__contains", dev.Name)
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Alarm
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func AddAlarm(ob *models.Alarm) error {
	if len(ob.Name) == 0 {
		return errors.New("name not be empty")
	}
	rs, err := GetAlarm(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("alarm is exist")
	}
	ob.State = models.Stopped
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = time.Now()
	_, err = o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateAlarm(ob *models.Alarm) error {
	//更新数据
	o := orm.NewOrm()
	var columns []string
	if len(ob.Name) > 0 {
		columns = append(columns, "Name")
	}
	if len(ob.Desc) > 0 {
		columns = append(columns, "Desc")
	}
	if len(ob.Triggers) > 0 {
		columns = append(columns, "Triggers")
	}
	if len(ob.Actions) > 0 {
		columns = append(columns, "Actions")
	}
	if len(columns) == 0 {
		return errors.New("no data to update")
	}
	_, err := o.Update(ob, columns...)
	if err != nil {
		return err
	}
	return nil
}

// 更新在线状态
func UpdateAlarmStatus(state string, id int64) error {
	if id == 0 {
		return errors.New("id not be empty")
	}
	if len(state) == 0 {
		return errors.New("state not be empty")
	}
	o := orm.NewOrm()
	var ob = &models.Alarm{Id: id, State: state}
	_, err := o.Update(ob, "State")
	if err != nil {
		return err
	}
	return nil
}

func DeleteAlarm(ob *models.Alarm) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		return err
	}
	return nil
}

func GetAlarm(alarmId int64) (*models.Alarm, error) {
	o := orm.NewOrm()
	p := models.Alarm{Id: alarmId}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}

func GetAlarmList(q models.Alarm) ([]models.Alarm, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(models.AlarmLog{})
	if q.Target == "device" {
		qs = qs.Filter("TargetId", q.TargetId)
	}
	if q.Target == "product" {
		qs = qs.Filter("TargetId", q.TargetId)
	}
	var result []models.Alarm
	_, err := qs.All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func PageAlarmLog(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.AlarmLog
	err := json.Unmarshal(page.Condition, &dev)
	if err != nil {
		return nil, err
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.AlarmLog{})

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.AlarmLog
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func GetAlarmLog(q models.AlarmLog) ([]models.AlarmLog, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(models.AlarmLog{})
	if len(q.DeviceId) > 0 {
		qs = qs.Filter("DeviceId", q.DeviceId)
	}
	if len(q.ProductId) > 0 {
		qs = qs.Filter("ProductId", q.ProductId)
	}
	var result []models.AlarmLog
	_, err := qs.All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func SolveAlarmLog(q models.AlarmLog) error {
	if q.Id == 0 {
		return errors.New("id not be empty")
	}
	q.State = "solve"
	o := orm.NewOrm()
	_, err := o.Update(q, "State")
	if err != nil {
		return err
	}
	return nil
}
