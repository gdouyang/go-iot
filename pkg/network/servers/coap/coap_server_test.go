package coapserver_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	coapserver "go-iot/pkg/network/servers/coap"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/timeseries"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
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
	t.Cleanup(func() { _ = srv.Stop() })
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

// TestCoapServer_ConcurrentRequests 覆盖并发 UDP 请求：
// coap 是无状态模型，高并发请求不得 panic、数据不得丢失
func TestCoapServer_ConcurrentRequests(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeUDPPort(t)
	productId := fmt.Sprintf("coap-conc-%d", port)
	testhelper.InitCore()
	core.DeleteProduct(productId)
	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK, testhelper.DefaultTSL())
	require.NoError(t, err)
	require.NoError(t, core.PutProduct(product))
	for i := 0; i < 20; i++ {
		require.NoError(t, core.PutDevice(core.NewDevice(fmt.Sprintf("coap-conc-%d", i), productId, 0)))
	}

	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/test"}]}`,
	}
	srv := coapserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() { _ = srv.Stop() })
	_, err = core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	const n = 20
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			deviceId := fmt.Sprintf("coap-conc-%d", i)
			co, err := udp.Dial(fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				errCh <- err
				return
			}
			defer co.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			body := fmt.Sprintf(`{"deviceId":"%s","temperature":%d}`, deviceId, i)
			res, err := co.Post(ctx, "/test?deviceId="+deviceId, message.AppJSON, strings.NewReader(body))
			if err != nil {
				errCh <- err
				return
			}
			if res.Code() != codes.Content {
				errCh <- fmt.Errorf("req %d: code=%s", i, res.Code().String())
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent request error: %v", err)
	}
	// 20 个请求全部落数
	require.EqualValues(t, n, timeseries.MockPropertiesCount())
}
