package orm_test

import (
	"go-iot/pkg/core/es/orm"
	"go-iot/pkg/models"
	"testing"
)

func TestDbInit(t *testing.T) {
	orm.RegisterModel(
		//new(models.User), new(models.Role), new(models.UserRelRole),
		// new(models.MenuResource), new(models.AuthResource), new(models.SystemConfig),
		new(models.Product), //new(models.Device), new(models.Network),
	// new(models.Rule), new(models.RuleRelDevice), new(models.AlarmLog),
	// new(models.Notify),
	)
}

func Test2(t *testing.T) {
	orm.RegisterModel(
		//new(models.User), new(models.Role), new(models.UserRelRole),
		// new(models.MenuResource), new(models.AuthResource), new(models.SystemConfig),
		new(models.Product), //new(models.Device), new(models.Network),
	// new(models.Rule), new(models.RuleRelDevice), new(models.AlarmLog),
	// new(models.Notify),
	)
}
