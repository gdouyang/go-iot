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
		return errors.New("Alarm is exist")
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
	var ob models.Alarm = models.Alarm{Id: id, State: state}
	o := orm.NewOrm()
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

func GetAlarm(AlarmId int64) (*models.Alarm, error) {
	o := orm.NewOrm()
	p := models.Alarm{Id: AlarmId}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}
