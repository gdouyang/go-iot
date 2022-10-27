package base

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 分页查询设备
func ListUser(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var n models.User
	err1 := json.Unmarshal(page.Condition, &n)
	if err1 != nil {
		return nil, err1
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.User{})

	id := n.Id
	if id > 0 {
		qs = qs.Filter("id", id)
	}
	if len(n.Username) > 0 {
		qs = qs.Filter("name__contains", n.Username)
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.User
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func AddUser(ob *models.User) error {
	rs, err := GetUserByEntity(models.User{Username: ob.Username})
	if err != nil {
		return err
	}
	if rs.Id > 0 {
		return errors.New("username已存在")
	}
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = time.Now()
	_, err = o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUser(ob *models.User) error {
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "Nickname")
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func UpdateUserPwd(ob *models.User) error {
	if ob.Id == 0 {
		return errors.New("id not be empty")
	}
	if len(ob.Username) == 0 {
		return errors.New("username not be empty")
	}
	data := []byte(ob.Username + ob.Password)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has) //将[]byte转成16进制
	ob.Password = md5str
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "Password")
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserEnable(ob *models.User) error {
	if ob.Id == 0 {
		return errors.New("id not be empty")
	}
	o := orm.NewOrm()
	_, err := o.Update(ob, "enableFlag")
	if err != nil {
		return err
	}
	return nil
}

func DeleteUser(ob *models.User) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		logs.Error("delete fail", err)
		return err
	}
	return nil
}

func GetUser(id int64) (*models.User, error) {

	o := orm.NewOrm()

	p := models.User{Id: id}
	err := o.Read(&p, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}

func GetUserByEntity(p models.User) (*models.User, error) {

	o := orm.NewOrm()
	cols := []string{}
	if p.Id != 0 {
		cols = append(cols, "id")
	}
	if len(p.Username) > 0 {
		cols = append(cols, "username")
	}
	err := o.Read(&p, cols...)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}
