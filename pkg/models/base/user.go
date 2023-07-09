package base

import (
	"crypto/md5"
	"errors"
	"fmt"
	"go-iot/pkg/boot"
	"go-iot/pkg/models"

	"go-iot/pkg/es/orm"

	logs "go-iot/pkg/logger"
)

func init() {
	boot.AddStartLinstener(func() {
		admin, _ := GetUser(1)
		if admin == nil {
			AddUser(&UserDTO{
				User: models.User{
					Id:         1,
					Username:   "admin",
					Nickname:   "admin",
					Password:   "123456",
					EnableFlag: true,
				},
			})
			logs.Infof("init admin user")
		}
	})
}

type UserDTO struct {
	models.User
	RoleId int64 `json:"roleId"`
}

// 分页查询设备
func PageUser(page *models.PageQuery, createId int64) (*models.PageResult[models.User], error) {
	var pr *models.PageResult[models.User]
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.User{})
	qs = qs.FilterTerm(page.Condition...)
	qs = qs.Filter("createId", createId)
	qs.SearchAfter = page.SearchAfter
	var result []models.User
	_, err := qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime", "-id").All(&result)
	if err != nil {
		return nil, err
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}
	for _, us := range result {
		us.Password = ""
	}
	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	p.SearchAfter = qs.LastSort
	pr = &p

	return pr, nil
}

func AddUser(ob *UserDTO) error {
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
	u := &ob.User
	Md5Pwd(u)
	//插入数据
	o := orm.NewOrm()
	u.CreateTime = models.NewDateTime()
	_, err = o.Insert(u)
	if err != nil {
		return err
	}
	if ob.RoleId > 0 {
		err = AddUserRelRole(u.Id, ob.RoleId)
		if err != nil {
			o.Delete(u)
			return err
		}
	}
	return nil
}

func UpdateUser(ob *UserDTO) error {
	if len(ob.Nickname) == 0 {
		return fmt.Errorf("nickname must be present")
	}
	u := &ob.User
	o := orm.NewOrm()
	_, err := o.Update(u, "Nickname", "Email", "Desc")
	if err != nil {
		return err
	}
	DeleteUserRelRoleByUserId(u.Id)
	err = AddUserRelRole(u.Id, ob.RoleId)
	if err != nil {
		o.Delete(u)
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
		logs.Errorf("delete fail %v", err)
		return err
	}
	DeleteUserRelRoleByUserId(ob.Id)
	return nil
}

func GetUser(id int64) (*UserDTO, error) {

	o := orm.NewOrm()

	p := models.User{Id: id}
	err := o.Read(&p, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		dto := &UserDTO{User: p}
		list, err := GetUserRelRoleByUserId(id)
		if err != nil {
			logs.Errorf("GetUserRelRoleByUserId error: %v", err)
		}
		if len(list) > 0 {
			dto.RoleId = list[0].RoleId
		}
		return dto, nil
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
