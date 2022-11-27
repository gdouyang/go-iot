package scene

import (
	"encoding/json"
	"errors"
	"go-iot/codec/eventbus"
	"go-iot/models"
	"go-iot/ruleengine"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func init() {
	eventbus.Subscribe(eventbus.GetAlarmTopic("*", "*"), saveAlarmEvent)
}

func saveAlarmEvent(data interface{}) {
	if data == nil {
		return
	}
	switch t := data.(type) {
	case ruleengine.AlarmEvent:
		b, err := json.Marshal(t.Data)
		if err != nil {
			logs.Error(err)
			return
		}
		log := models.AlarmLog{
			ProductId: t.ProductId,
			DeviceId:  t.DeviceId,
			RuleId:    t.RuleId,
			AlarmName: t.AlarmName,
			AlarmData: string(b),
		}
		go AddAlarmLog(log)
	}
}

func PageAlarmLog(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var q models.AlarmLog
	err := json.Unmarshal(page.Condition, &q)
	if err != nil {
		return nil, err
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.AlarmLog{})
	if len(q.DeviceId) > 0 {
		qs = qs.Filter("DeviceId", q.DeviceId)
	}
	if len(q.ProductId) > 0 {
		qs = qs.Filter("ProductId", q.ProductId)
	}
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

func AddAlarmLog(q models.AlarmLog) error {
	o := orm.NewOrm()
	q.CreateTime = time.Now()
	_, err := o.Insert(q)
	if err != nil {
		return err
	}
	return nil
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
