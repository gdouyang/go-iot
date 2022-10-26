package base

import (
	"go-iot/models"

	"github.com/beego/beego/v2/client/orm"
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
