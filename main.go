package main

import (
	_ "go-iot/api"

	"github.com/beego/beego/v2/server/web"
)

func main() {
	web.Run()
}
