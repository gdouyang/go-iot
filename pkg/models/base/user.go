package base

import (
	"crypto/md5"
	"errors"
	"fmt"
	"go-iot/pkg/core/boot"
	"go-iot/pkg/models"

	"github.com/beego/beego/v2/client/orm"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	boot.AddStartLinstener(func() {
		admin, _ := GetUser(1)
		if admin == nil {
			AddUser(&models.User{
				Id:         1,
				Username:   "admin",
				Nickname:   "admin",
				Password:   "123456",
				EnableFlag: true,
			})
			logs.Info("init admin user")
		}
	})
}

// 分页查询设备
func PageUser(page *models.PageQuery[models.User], createId int64) (*models.PageResult[models.User], error) {
	var pr *models.PageResult[models.User]
	var n models.User = page.Condition

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.User{})

	id := n.Id
	if id > 0 {
		qs = qs.Filter("id", id)
	}
	if len(n.Username) > 0 {
		qs = qs.Filter("username__contains", n.Username)
	}
	if len(n.Nickname) > 0 {
		qs = qs.Filter("nickname__contains", n.Nickname)
	}
	qs = qs.Filter("createId", createId)

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.User
	_, err = qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime").All(&result)
	if err != nil {
		return nil, err
	}
	for _, us := range result {
		us.Password = ""
	}
	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func AddUser(ob *models.User) error {
	if len(ob.Password) == 0 {
		return errors.New("password must be present")
	}
	rs, err := GetUserByEntity(models.User{Username: ob.Username})
	if err != nil {
		return err
	}
	if rs != nil {
		return errors.New("user exist")
	}
	Md5Pwd(ob)
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = models.NewDateTime()
	_, err = o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUser(ob *models.User) error {
	if len(ob.Nickname) == 0 {
		return fmt.Errorf("nickname must be present")
	}
	o := orm.NewOrm()
	_, err := o.Update(ob, "Nickname", "Email", "Desc")
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserPwd(ob *models.User) error {
	if ob.Id == 0 {
		return errors.New("id must be present")
	}
	if len(ob.Username) == 0 {
		return errors.New("username must be present")
	}
	Md5Pwd(ob)
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "Password")
	if err != nil {
		return err
	}
	return nil
}

func Md5Pwd(ob *models.User) {
	data := []byte(ob.Username + ob.Password)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has) //将[]byte转成16进制
	ob.Password = md5str
}

func UpdateUserEnable(ob *models.User) error {
	if ob.Id == 0 {
		return errors.New("id must be present")
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
		return &p, err
	}
}
