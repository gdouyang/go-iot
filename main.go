package main

import (
	_ "go-iot/controllers"

	_ "go-iot/agent"
	_ "go-iot/provider/shunzhou"
	_ "go-iot/provider/xixun"
	_ "go-iot/provider/xixun/controllers"

	"github.com/astaxie/beego"
)

func main() {
	beego.Run()
}
