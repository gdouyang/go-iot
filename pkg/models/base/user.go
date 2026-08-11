package base

import (
	"errors"
	"fmt"
	"go-iot/pkg/models"

	"go-iot/pkg/es/orm"

	logs "go-iot/pkg/logger"
)

// DefaultAdminPassword 未配置时的初始密码（仅首次创建 admin；生产请配置 admin.password / GOIOT_ADMIN_PASSWORD）。
const DefaultAdminPassword = "123456"

// EnsureDefaultAdmin 若不存在 id=1 的用户则创建默认 admin（由 app.Start 显式调用）。
// password 为空时使用 DefaultAdminPassword。仅首次创建生效，已存在用户不会改密。
// 密码以 bcrypt 入库。
func EnsureDefaultAdmin(password string) {
	admin, _ := GetUser(1)
	if admin != nil {
		return
	}
	usedDefault := false
	if len(password) == 0 {
		password = DefaultAdminPassword
		usedDefault = true
	}
	if err := AddUser(&UserDTO{
		User: models.User{
			Id:         1,
			Username:   "admin",
			Nickname:   "admin",
			Password:   password,
			EnableFlag: true,
		},
	}); err != nil {
		logs.Errorf("init admin user error: %v", err)
		return
	}
	if usedDefault {
		logs.Warnf("init admin user with DEFAULT password %q — set admin.password or GOIOT_ADMIN_PASSWORD for production", DefaultAdminPassword)
	} else {
		logs.Infof("init admin user (password from config)")
	}
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
	hash, err := HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hash
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

func UpdateUserBaseInfo(ob *UserDTO) error {
	if len(ob.Nickname) == 0 {
		return fmt.Errorf("nickname must be present")
	}
	u := &ob.User
	o := orm.NewOrm()
	_, err := o.Update(u, "Nickname", "Email", "Desc")
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
	if len(ob.Password) == 0 {
		return errors.New("password must be present")
	}
	// 已是 bcrypt 则直接写；否则对明文做 bcrypt（改密 / 懒迁移）
	if !IsBcryptHash(ob.Password) {
		hash, err := HashPassword(ob.Password)
		if err != nil {
			return err
		}
		ob.Password = hash
	}
	o := orm.NewOrm()
	_, err := o.Update(ob, "Password")
	if err != nil {
		return err
	}
	return nil
}

// Md5Pwd 已废弃：请使用 HashPassword / CheckPassword。
// 保留空实现会破坏调用方；改为写入 bcrypt，兼容仍调用 Md5Pwd 的旧代码路径。
// Deprecated: use HashPassword.
func Md5Pwd(ob *models.User) {
	if ob == nil || len(ob.Password) == 0 {
		return
	}
	if IsBcryptHash(ob.Password) {
		return
	}
	hash, err := HashPassword(ob.Password)
	if err != nil {
		logs.Errorf("Md5Pwd(HashPassword) error: %v", err)
		return
	}
	ob.Password = hash
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
