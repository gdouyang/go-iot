package base

import (
	"go-iot/pkg/models"

	"go-iot/pkg/core/es/orm"
)

func GetUserRelRoleByUserId(userId int64) ([]models.UserRelRole, error) {
	var reslut []models.UserRelRole
	o := orm.NewOrm()
	qs := o.QueryTable(&models.UserRelRole{})
	qs = qs.Filter("userId", userId)

	_, err := qs.All(&reslut)

	if err != nil {
		return nil, err
	}

	return reslut, nil
}

func AddUserRelRole(userId, roleId int64) error {
	o := orm.NewOrm()
	ob := &models.UserRelRole{UserId: userId, RoleId: roleId}
	_, err := o.Insert(ob)
	return err
}

func DeleteUserRelRoleByUserId(userId int64) error {
	o := orm.NewOrm()
	ob := &models.UserRelRole{UserId: userId}
	_, err := o.Delete(ob, "userId")
	return err
}
