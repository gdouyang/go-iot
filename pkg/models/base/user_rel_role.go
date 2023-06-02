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
