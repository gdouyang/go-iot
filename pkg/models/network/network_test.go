package network

import (
	"fmt"
	"testing"

	"go-iot/pkg/models"

	logs "go-iot/pkg/logger"
)

// 查询依赖 ES。默认跳过，需要联调时去掉 t.Skip 即可。
func TestGetUnuseNetwork(t *testing.T) {
	logs.InitNop()
	t.Skip("requires Elasticsearch; remove this Skip to run manually")

	models.RegisterModels()

	nw, err := GetUnuseNetwork()
	if err != nil {
		fmt.Println(err)
	}
	if nw != nil {
		fmt.Println(nw.Id)
	}
}
