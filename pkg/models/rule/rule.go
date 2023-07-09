package rule

import (
	"errors"
	"go-iot/pkg/models"

	"go-iot/pkg/es/orm"
)

// 分页查询
func PageRule(page *models.PageQuery, createId *int64) (*models.PageResult[models.Rule], error) {
	var pr *models.PageResult[models.Rule]

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Rule{})

	qs.FilterTerm(page.Condition...)
	if createId != nil {
		qs = qs.Filter("CreateId", *createId)
	}
	qs.SearchAfter = page.SearchAfter

	var cols = []string{"Id", "Name", "State", "Desc", "CreateId", "CreateTime"}
	var result []models.Rule
	_, err := qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime", "-id").All(&result, cols...)
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

func AddRule(ob *models.RuleModel) error {
	if len(ob.Name) == 0 {
		return errors.New("name must be present")
	}
	rs, err := GetRule(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("scene is exist")
	}
	if len(ob.DeviceIds) > 50 {
		return errors.New("size of deviceIds must less 51")
	}
	ob.State = models.Stopped
	en := ob.ToEnitty()
	//插入数据
	o := orm.NewOrm()
	en.CreateTime = models.NewDateTime()
	_, err = o.Insert(&en)
	if err != nil {
		return err
	}
	list := []*models.RuleRelDevice{}
	for _, deviceId := range ob.DeviceIds {
		list = append(list, &models.RuleRelDevice{
			RuleId:   en.Id,
			DeviceId: deviceId,
		})
	}
	if len(list) > 0 {
		_, err = o.InsertMulti(10, &list)
		return err
	}
	return nil
}

func UpdateRule(ob *models.RuleModel) error {
	//更新数据
	en := ob.ToEnitty()
	var columns []string
	if len(ob.Name) > 0 {
		columns = append(columns, "Name")
	}
	if len(ob.TriggerType) > 0 {
		columns = append(columns, "TriggerType")
	}
	if len(ob.ProductId) > 0 {
		columns = append(columns, "ProductId")
	}
	if len(en.Trigger) > 0 {
		columns = append(columns, "Trigger")
	}
	if len(ob.Actions) > 0 {
		columns = append(columns, "Actions")
	}
	if len(ob.Type) > 0 {
		columns = append(columns, "Type")
	}
	if len(ob.Desc) > 0 {
		columns = append(columns, "Desc")
	}
	if len(columns) == 0 {
		return errors.New("no data to update")
	}
	if len(ob.DeviceIds) > 50 {
		return errors.New("size of deviceIds must less 51")
	}
	o := orm.NewOrm()
	_, err := o.Update(&en, columns...)
	if err != nil {
		return err
	}
	srd := models.RuleRelDevice{
		RuleId: en.Id,
	}
	_, err = o.Delete(&srd, "RuleId")
	if err != nil {
		return err
	}
	list := []*models.RuleRelDevice{}
	for _, deviceId := range ob.DeviceIds {
		list = append(list, &models.RuleRelDevice{
			RuleId:   en.Id,
			DeviceId: deviceId,
		})
	}
	if len(list) > 0 {
		_, err = o.InsertMulti(10, &list)
		return err
	}
	return nil
}

// 更新在线状态
func UpdateRuleStatus(state string, id int64) error {
	if id == 0 {
		return errors.New("id must be present")
	}
	if len(state) == 0 {
		return errors.New("state must be present")
	}
	o := orm.NewOrm()
	var ob = &models.Rule{Id: id, State: state}
	_, err := o.Update(ob, "State")
	if err != nil {
		return err
	}
	return nil
}

func DeleteRule(ob *models.Rule) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		return err
	}
	return nil
}

func GetRule(sceneId int64) (*models.RuleModel, error) {
	o := orm.NewOrm()
	p := models.Rule{Id: sceneId}
	err := o.Read(&p, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		m := models.RuleModel{}
		m.FromEnitty(p)
		//
		o := orm.NewOrm()
		qs := o.QueryTable(new(models.RuleRelDevice)).Filter("RuleId", p.Id)
		var cols = []string{"Id", "RuleId", "DeviceId"}
		var result []models.RuleRelDevice
		_, err = qs.All(&result, cols...)
		if err != nil {
			return nil, err
		}
		for _, rel := range result {
			m.DeviceIds = append(m.DeviceIds, rel.DeviceId)
		}
		return &m, nil
	}
}

func GetRuleMust(id int64) (*models.RuleModel, error) {
	p, err := GetRule(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("scene not exist")
	}
	return p, nil
}
