package httpserver_test

import (
	"go-iot/pkg/codec"
	"go-iot/pkg/codec/tsl"
	"go-iot/pkg/models"
	_ "go-iot/pkg/models/device"
	httpserver "go-iot/pkg/network/servers/http"
	"io"
	"log"
	"net/http"
	"net/url"
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
  context.SaveProperties(data)
	context.GetSession().Response(data)
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context))
}
`

var network codec.NetworkConf = codec.NetworkConf{
	Name:      "test server",
	ProductId: "test-product",
	CodecId:   "script_codec",
	Port:      18080,
	Script:    script,
}

func init() {
	models.DefaultDbConfig.Url = "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"
	models.InitDb()
	var product *codec.Product = &codec.Product{
		Id:          "test-product",
		Config:      make(map[string]string),
		StorePolicy: "mock",
	}
	tslData := &tsl.TslData{}
	err := tslData.FromJson(`{"properties":[{"id":"temperature","valueType":{"type":"float"}}],"functions":[{"id":"func1","inputs":[{"id":"name", "valueType":{"type":"string"}}]}]}`)
	if err != nil {
		logs.Error(err)
	}
	product.TslData = tslData
	codec.PutProduct(product)
	{
		device := &codec.Device{
			Id:        "1234",
			ProductId: product.GetId(),
			Data:      make(map[string]string),
			Config:    make(map[string]string),
		}
		codec.PutDevice(device)
	}
}

func TestServer(t *testing.T) {
	network := network
	network.Configuration = `{"host": "localhost", "useTLS": false, "paths":["/test"]}`
	httpserver.NewServer().Start(network)
	codec.NewCodec(network)
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
	// time.Sleep(time.Second * 11)
}

func TestHttp(t *testing.T) {
	u, err := url.ParseRequestURI("http://www.baidu.com")
	if err != nil {
		logs.Error(err)
		return
	}
	client := http.Client{Timeout: time.Second * 3}
	var req *http.Request = &http.Request{
		Method: "get",
		URL:    u,
		Header: map[string][]string{},
	}
	resp, err := client.Do(req)
	if err != nil {
		logs.Error(err)
		return
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Info(string(b))
}

func TestHttp1(t *testing.T) {
	resp := codec.HttpRequest(map[string]interface{}{
		"method": "get",
		"url":    "http://www.baidu.com",
	})
	logs.Info(resp)
}
