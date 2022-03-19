package main

import (
	_ "go-iot/provider/north"

	"github.com/beego/beego/v2/server/web"
)

func main() {
	web.Run()
}
