package scene

import (
	"encoding/json"
	"errors"
	"go-iot/models"

	"github.com/beego/beego/v2/client/orm"
)

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
