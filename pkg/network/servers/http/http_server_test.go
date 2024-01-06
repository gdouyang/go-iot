package httpserver_test

import (
	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	httpserver "go-iot/pkg/network/servers/http"
	"go-iot/pkg/store"
	_ "go-iot/pkg/timeseries"
	"go-iot/pkg/tsl"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	logs "go-iot/pkg/logger"
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

var network1 network.NetworkConf = network.NetworkConf{
	Name:      "test server",
	ProductId: "test-product",
	CodecId:   "script_codec",
	Port:      18080,
	Script:    script,
}

func init() {
	logs.InitNop()
	core.RegDeviceStore(store.NewMockDeviceStore())
	var product *core.Product = &core.Product{
		Id:          "test-product",
		Config:      make(map[string]string),
		StorePolicy: "mock",
	}
	tslData := &tsl.TslData{}
	err := tslData.FromJson(`{"properties":[{"id":"temperature","type":"float"}],
	"functions":[{"id":"func1","inputs":[{"id":"name", "type":"string"}]}]}`)
	if err != nil {
		logs.Errorf(err.Error())
	}
	product.TslData = tslData
	core.PutProduct(product)
	{
		device := core.NewDevice("1234", product.Id, 0)
		core.PutDevice(device)
	}
}

func TestServer(t *testing.T) {
	network := network1
	network.Configuration = `{"host": "localhost", "useTLS": false, "paths":["/test"]}`
	httpserver.NewServer().Start(network)
	core.NewCodec(network1.CodecId, network1.ProductId, network1.Script)
	initClient()
}

func initClient() {
	res, err := http.Post("http://localhost:18080/test", "application/json", strings.NewReader(`{"deviceId": "1234", "temperature": 16.1}`))
	//Get请求
	// res, err := http.Get("http://www.baidu.com")
	if err != nil {
		logs.Errorf(err.Error())
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
		logs.Errorf(err.Error())
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
		logs.Errorf(err.Error())
		return
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Errorf(err.Error())
		return
	}
	logs.Infof(string(b))
}

func TestHttp1(t *testing.T) {
	resp := core.HttpRequest(map[string]interface{}{
		"method": "get",
		"url":    "http://www.baidu.com",
	})
	logs.Infof("%v", resp)
}
