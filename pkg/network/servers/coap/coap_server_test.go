package coapserver_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	coapserver "go-iot/pkg/network/servers/coap"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/timeseries"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/stretchr/testify/require"
)

const script = `
function OnMessage(context) {
	context.DeviceOnline(context.GetQuery("deviceId"))
	var data = JSON.parse(context.MsgToString())
	context.SaveProperties(data)
	context.GetSession().Response(JSON.stringify({ok:true}))
}
function OnInvoke(context) {}
`

func TestCoapServer_DeviceOnlineAndSaveProperties(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeUDPPort(t)
	productId := fmt.Sprintf("coap-p-%d", port)
	deviceId := "dev-coap-1"
	testhelper.SetupProductDevice(t, productId, deviceId)

	conf := network.NetworkConf{
		Name:          "coap-it",
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/test"}]}`,
	}
	srv := coapserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() {
		// Coap Stop 可能在 server 未赋值时 panic，做保护
		defer func() { _ = recover() }()
		_ = srv.Stop()
	})
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	before := timeseries.MockPropertiesCount()
	co, err := udp.Dial(fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	body := fmt.Sprintf(`{"deviceId":"%s","temperature":16.1}`, deviceId)
	res, err := co.Post(ctx, "/test?deviceId="+deviceId, message.AppJSON, strings.NewReader(body))
	require.NoError(t, err)
	t.Logf("coap response: %s %s", res.Code().String(), res.String())

	require.Greater(t, timeseries.MockPropertiesCount(), before)
	saved := timeseries.MockLastProperties()
	require.NotNil(t, saved)
	require.Equal(t, deviceId, fmt.Sprint(saved["deviceId"]))
	require.EqualValues(t, 16.1, saved["temperature"])
}

func TestCoapServerSpec_FromJson(t *testing.T) {
	// 使用 package 外无法访问 unexported，通过 Start 错误配置覆盖
	srv := coapserver.NewServer()
	err := srv.Start(network.NetworkConf{
		ProductId:     "coap-bad",
		Port:          1,
		Configuration: `{bad`,
	})
	require.Error(t, err)
}
