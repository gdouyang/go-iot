package orm_test

import (
	"testing"

	"go-iot/pkg/es/orm"
	"go-iot/pkg/models"

	logs "go-iot/pkg/logger"
)

// RegisterModel 会访问 ES 建索引模板。默认跳过，需要联调时去掉 t.Skip 即可。
func TestDbInit(t *testing.T) {
	logs.InitNop()
	t.Skip("requires Elasticsearch; remove this Skip to run manually")

	orm.RegisterModel(
		//new(models.User), new(models.Role), new(models.UserRelRole),
		// new(models.MenuResource), new(models.AuthResource), new(models.SystemConfig),
		new(models.Product), //new(models.Device), new(models.Network),
	// new(models.Rule), new(models.RuleRelDevice), new(models.AlarmLog),
	// new(models.Notify),
	)
}

func Test2(t *testing.T) {
	logs.InitNop()
	t.Skip("requires Elasticsearch; remove this Skip to run manually")

	orm.RegisterModel(
		//new(models.User), new(models.Role), new(models.UserRelRole),
		// new(models.MenuResource), new(models.AuthResource), new(models.SystemConfig),
		new(models.Product), //new(models.Device), new(models.Network),
	// new(models.Rule), new(models.RuleRelDevice), new(models.AlarmLog),
	// new(models.Notify),
	)
}
