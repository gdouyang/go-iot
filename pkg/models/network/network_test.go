package network

import (
	"fmt"
	"go-iot/pkg/models"
	"testing"
)

func TestGetUnuseNetwork(t *testing.T) {
	models.InitDb()

	nw, err := GetUnuseNetwork()
	if err != nil {
		fmt.Println(err)
	}
	if nw != nil {
		fmt.Println(nw.Id)
	}
}
