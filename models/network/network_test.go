package network

import (
	"fmt"
	"go-iot/models"
	"testing"

	"github.com/beego/beego/v2/client/orm"
)

func TestGetUnuseNetwork(t *testing.T) {
	orm.Debug = true
	models.DefaultDbConfig.Url = "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"
	models.InitDb()

	nw, err := GetUnuseNetwork()
	if err != nil {
		fmt.Println(err)
	}
	if nw != nil {
		fmt.Println(nw.Id)
	}
}
