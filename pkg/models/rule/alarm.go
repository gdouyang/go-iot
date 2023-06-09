package rule

import (
	"encoding/json"
	"errors"
	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/models"
	"go-iot/pkg/ruleengine"

	"go-iot/pkg/es/orm"

	logs "go-iot/pkg/logger"
)

func init() {
	eventbus.Subscribe(eventbus.GetAlarmTopic("*", "*"), saveAlarmEvent)
}

func saveAlarmEvent(data eventbus.Message) {
	if data == nil {
		return
	}
	if t, ok := data.(*ruleengine.AlarmEvent); ok {
		b, err := json.Marshal(t.Data)
		if err != nil {
			logs.Errorf(err.Error())
			return
		}
		device := core.GetDevice(t.DeviceId)
		if device == nil {
			logs.Errorf("saveAlarmEvent error: device not found")
			return
		}
		log := models.AlarmLog{
			ProductId: t.ProductId,
			DeviceId:  t.DeviceId,
			RuleId:    t.RuleId,
			CreateId:  device.GetCreateId(),
			AlarmName: t.AlarmName,
			AlarmData: string(b),
			State:     "unsolve",
		}
		go AddAlarmLog(log)
	}
}

func PageAlarmLog(page *models.PageQuery, createId int64) (*models.PageResult[models.AlarmLog], error) {
	var pr *models.PageResult[models.AlarmLog]
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.AlarmLog{})
	qs = qs.FilterTerm(page.Condition...)
	qs = qs.Filter("CreateId", createId)
	qs.SearchAfter = page.SearchAfter
	var result []models.AlarmLog
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

func AddAlarmLog(q models.AlarmLog) error {
	o := orm.NewOrm()
	q.CreateTime = models.NewDateTime()
	_, err := o.Insert(&q)
	if err != nil {
		return err
	}
	return nil
}

func SolveAlarmLog(q models.AlarmLog) error {
	if q.Id == 0 {
		return errors.New("id must be present")
	}
	q.State = "solve"
	o := orm.NewOrm()
	_, err := o.Update(q, "State", "desc")
	if err != nil {
		return err
	}
	return nil
}
