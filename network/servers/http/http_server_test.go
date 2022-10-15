package httpserver_test

import (
	"go-iot/codec"
	"go-iot/models"
	_ "go-iot/models/device"
	httpserver "go-iot/network/servers/http"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

const script = `
function OnMessage(context) {
	console.log("OnMessage: " + context.MsgToString())
	context.DeviceOnline(context.GetQuery("deviceId"))
  var data = JSON.parse(context.MsgToString())
  context.Save(data)
	context.GetSession().Send(data)
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context))
}
`

var network codec.Network = codec.Network{
	Name:      "test server",
	ProductId: "test123",
	CodecId:   "script_codec",
	Port:      18080,
	Script:    script,
}

func TestServer(t *testing.T) {
	models.DefaultDbConfig.Url = "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"
	models.InitDb()

	network := network
	network.Configuration = `{"host": "localhost", "useTLS": false, "paths":["/test"]}`
	httpserver.ServerStart(network)
	initClient()
}

func initClient() {
	res, err := http.Post("http://localhost:18080/test", "application/json", strings.NewReader(`{"deviceId": "1234", "temperature": 16.1}`))
	//Get请求
	// res, err := http.Get("http://www.baidu.com")
	if err != nil {
		logs.Error(err)
	}
	//利用ioutil包读取百度服务器返回的数据
	data, err := io.ReadAll(res.Body)
	res.Body.Close() //一定要记得关闭连接
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("client Received: %d %s \n", res.StatusCode, data)
	time.Sleep(time.Second * 11)
}
