package scene

import (
	"encoding/json"
	"errors"
	"go-iot/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// 分页查询
func ListScene(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.Scene
	err := json.Unmarshal(page.Condition, &dev)
	if err != nil {
		return nil, err
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Scene{})

	if len(dev.Name) > 0 {
		qs = qs.Filter("name__contains", dev.Name)
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}
	var cols = []string{"Id", "Name", "State", "Desc", "CreateId", "CreateTime"}
	var result []models.Scene
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result, cols...)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func AddScene(ob *models.Scene) error {
	if len(ob.Name) == 0 {
		return errors.New("name not be empty")
	}
	rs, err := GetScene(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("scene is exist")
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

func UpdateScene(ob *models.Scene) error {
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
func UpdateSceneStatus(state string, id int64) error {
	if id == 0 {
		return errors.New("id not be empty")
	}
	if len(state) == 0 {
		return errors.New("state not be empty")
	}
	var ob models.Scene = models.Scene{Id: id, State: state}
	o := orm.NewOrm()
	_, err := o.Update(ob, "State")
	if err != nil {
		return err
	}
	return nil
}

func DeleteScene(ob *models.Scene) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		return err
	}
	return nil
}

func GetScene(SceneId int64) (*models.Scene, error) {
	o := orm.NewOrm()
	p := models.Scene{Id: SceneId}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}
