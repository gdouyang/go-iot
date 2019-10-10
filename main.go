package main

import (
	_ "go-iot/routers"

	_ "go-iot/agent"
	_ "go-iot/provider/shunzhou"
	_ "go-iot/provider/xixun"

	"github.com/astaxie/beego"
)

func main() {
	beego.Run()
}
