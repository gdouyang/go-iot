package coapserver_test

import (
	"context"
	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	coapserver "go-iot/pkg/network/servers/coap"
	"go-iot/pkg/store"
	_ "go-iot/pkg/timeseries"
	"go-iot/pkg/tsl"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/udp"

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
	Port:      5688,
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
	network.Configuration = `{"host": "", "useTLS": false, "paths":["/test"]}`
	coapserver.NewServer().Start(network)
	core.NewCodec(network1.CodecId, network1.ProductId, network1.Script)
	initClient()
}

func initClient() {
	co, err := udp.Dial("localhost:5688")
	if err != nil {
		log.Fatalf("Error dialing: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	res, err := co.Post(ctx, "/test?deviceId=1234", message.AppJSON, strings.NewReader(`{"deviceId": "1234", "temperature": 16.1}`))
	//Get请求
	// res, err := http.Get("http://www.baidu.com")
	if err != nil {
		panic(err)
	}
	log.Printf("client Received: %s %s \n", res.Code().String(), res.String())
	// time.Sleep(time.Second * 11)
}
