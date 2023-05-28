package rule

import (
	"encoding/json"
	"errors"
	"go-iot/pkg/core"
	"go-iot/pkg/core/eventbus"
	"go-iot/pkg/models"
	"go-iot/pkg/ruleengine"

	"github.com/beego/beego/v2/client/orm"

	"github.com/beego/beego/v2/core/logs"
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
			logs.Error(err)
			return
		}
		device := core.GetDevice(t.DeviceId)
		if device == nil {
			logs.Error("saveAlarmEvent error: device not found")
			return
		}
		log := models.AlarmLog{
			ProductId: t.ProductId,
			DeviceId:  t.DeviceId,
			RuleId:    t.RuleId,
			CreateId:  device.GetCreateId(),
			AlarmName: t.AlarmName,
			AlarmData: string(b),
		}
		go AddAlarmLog(log)
	}
}

func PageAlarmLog(page *models.PageQuery[models.AlarmLog]) (*models.PageResult[models.AlarmLog], error) {
	var pr *models.PageResult[models.AlarmLog]
	var q models.AlarmLog = page.Condition

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.AlarmLog{})
	if len(q.DeviceId) > 0 {
		qs = qs.Filter("DeviceId", q.DeviceId)
	}
	if len(q.ProductId) > 0 {
		qs = qs.Filter("ProductId", q.ProductId)
	}
	qs = qs.Filter("CreateId", q.CreateId)
	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.AlarmLog
	_, err = qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime").All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
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
	_, err := o.Update(q, "State")
	if err != nil {
		return err
	}
	return nil
}
