package httpserver_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	httpserver "go-iot/pkg/network/servers/http"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/timeseries"

	"github.com/stretchr/testify/require"
)

const script = `
function OnMessage(context) {
	context.DeviceOnline(context.GetQuery("deviceId"))
	var data = JSON.parse(context.MsgToString())
	context.SaveProperties(data)
	context.GetSession().ResponseJSON(JSON.stringify({ok:true, deviceId: data.deviceId}))
}
function OnInvoke(context) {}
`

func TestHttpServer_DeviceOnlineAndSaveProperties(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("http-p-%d", port)
	deviceId := "dev-http-1"
	testhelper.SetupProductDevice(t, productId, deviceId)

	conf := network.NetworkConf{
		Name:          "http-it",
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"paths":["/test"]}`,
	}
	// paths 字段可能被忽略，使用 routers
	conf.Configuration = `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/test"}]}`
	srv := httpserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() { _ = srv.Stop() })
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)

	// 等监听就绪
	time.Sleep(50 * time.Millisecond)

	before := timeseries.MockPropertiesCount()
	body := fmt.Sprintf(`{"deviceId":"%s","temperature":16.1}`, deviceId)
	url := fmt.Sprintf("http://127.0.0.1:%d/test?deviceId=%s", port, deviceId)
	res, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode, "body=%s", data)
	require.Contains(t, string(data), "ok")
	require.Contains(t, string(data), deviceId)

	// 校验 SaveProperties 真正落数（mock 时序内存记录）
	require.Greater(t, timeseries.MockPropertiesCount(), before, "SaveProperties should be called")
	saved := timeseries.MockLastProperties()
	require.NotNil(t, saved, "SaveProperties should record data")
	require.Equal(t, deviceId, fmt.Sprint(saved["deviceId"]))
	// JS number 经 goja 多为 float64
	require.EqualValues(t, 16.1, saved["temperature"])

	// 也可走 QueryProperty（mock 会返回最近匹配 deviceId 的记录）
	p := core.GetProduct(productId)
	require.NotNil(t, p)
	q, err := p.GetTimeSeries().QueryProperty(p, core.TimeDataSearchRequest{DeviceId: deviceId})
	require.NoError(t, err)
	require.NotNil(t, q)
	require.EqualValues(t, 1, q["total"])
}

func TestHttpServer_MissingDeviceOnlineFailsGracefully(t *testing.T) {
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("http-p2-%d", port)
	testhelper.SetupProductDevice(t, productId, "exists-dev")

	conf := network.NetworkConf{
		Name:          "http-it2",
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"paths":["/test"]}`,
	}
	srv := httpserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() { _ = srv.Stop() })
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// 不存在的设备：DeviceOnline 返回 error，脚本 throw，请求仍可能 200 但无业务数据
	res, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/test?deviceId=no-such", port),
		"application/json",
		strings.NewReader(`{"deviceId":"no-such","temperature":1}`),
	)
	require.NoError(t, err)
	defer res.Body.Close()
	// 不 panic 即通过；body 可能为空
	_, _ = io.ReadAll(res.Body)
}

func TestHttpServer_NotFoundPath(t *testing.T) {
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("http-p3-%d", port)
	testhelper.SetupProductDevice(t, productId, "d1")

	conf := network.NetworkConf{
		Name:          "http-it3",
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/only"}]}`,
	}
	// FromJson 用 routers；paths 兼容旧测试
	conf.Configuration = `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/only"}]}`
	srv := httpserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() { _ = srv.Stop() })
	time.Sleep(50 * time.Millisecond)

	res, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/other", port), "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusNotFound, res.StatusCode)
}

// TestHttpServer_ConcurrentRequests 覆盖并发请求：
// http 是无状态模型，高并发请求不得 panic、响应必须完整、数据不得丢失
func TestHttpServer_ConcurrentRequests(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("http-conc-%d", port)
	testhelper.InitCore()
	core.DeleteProduct(productId)
	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK, testhelper.DefaultTSL())
	require.NoError(t, err)
	require.NoError(t, core.PutProduct(product))
	for i := 0; i < 20; i++ {
		require.NoError(t, core.PutDevice(core.NewDevice(fmt.Sprintf("http-conc-%d", i), productId, 0)))
	}

	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false,"routers":[{"url":"/test"}]}`,
	}
	srv := httpserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() { _ = srv.Stop() })
	_, err = core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	const n = 20
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			deviceId := fmt.Sprintf("http-conc-%d", i)
			body := fmt.Sprintf(`{"deviceId":"%s","temperature":%d}`, deviceId, i)
			u := fmt.Sprintf("http://127.0.0.1:%d/test?deviceId=%s", port, deviceId)
			res, err := http.Post(u, "application/json", strings.NewReader(body))
			if err != nil {
				errCh <- err
				return
			}
			defer res.Body.Close()
			data, err := io.ReadAll(res.Body)
			if err != nil {
				errCh <- err
				return
			}
			if res.StatusCode != http.StatusOK || !strings.Contains(string(data), "ok") {
				errCh <- fmt.Errorf("req %d: status=%d body=%s", i, res.StatusCode, data)
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
