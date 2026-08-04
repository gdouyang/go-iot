package mqtt5_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	mqttserver "go-iot/pkg/network/servers/mqtt5"
	"go-iot/pkg/network/testhelper"
	_ "go-iot/pkg/timeseries"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/require"
)

func startMqtt5(t *testing.T, productId, script string) (port int32, cleanup func()) {
	t.Helper()
	port = testhelper.FreeTCPPort(t)
	testhelper.SetupProductDevice(t, productId, "dev-m5-ok")
	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false}`,
	}
	b := mqttserver.NewServer()
	require.NoError(t, b.Start(conf))
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)
	return port, func() { _ = b.Stop() }
}

func mqtt5Connect(t *testing.T, port int32, clientID string) (*autopaho.ConnectionManager, context.CancelFunc) {
	t.Helper()
	u, err := url.Parse(fmt.Sprintf("mqtt://127.0.0.1:%d", port))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: true,
		OnConnectError:                func(err error) { t.Logf("connect err: %v", err) },
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
		},
	}
	c, err := autopaho.NewConnection(ctx, cliCfg)
	require.NoError(t, err)
	require.NoError(t, c.AwaitConnection(ctx))
	return c, cancel
}

func TestMqtt5_ScriptError_DeviceOnlineUnknown(t *testing.T) {
	productId := fmt.Sprintf("m5-se-online-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	port, cleanup := startMqtt5(t, productId, script)
	defer cleanup()

	// ghost：DeviceOnline 失败后 broker 会拒绝连接（autopaho 会重试直到超时）
	u, err := url.Parse(fmt.Sprintf("mqtt://127.0.0.1:%d", port))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: true,
		OnConnectError:                func(err error) { t.Logf("ghost connect denied (expected): %v", err) },
		ClientConfig:                  paho.ClientConfig{ClientID: "ghost-m5"},
	}
	c, err := autopaho.NewConnection(ctx, cliCfg)
	require.NoError(t, err)
	err = c.AwaitConnection(ctx)
	// 预期失败或超时；无论哪种都不应有 session
	if err == nil {
		_ = c.Disconnect(context.Background())
	}
	require.Nil(t, core.GetSession("ghost-m5"))

	// 正确设备可上线
	c2, cancel2 := mqtt5Connect(t, port, "dev-m5-ok")
	defer cancel2()
	defer func() { _ = c2.Disconnect(context.Background()) }()
	time.Sleep(300 * time.Millisecond)
	require.NotNil(t, core.GetSession("dev-m5-ok"))
}

func TestMqtt5_ScriptError_SavePropertiesEmptyDeviceId(t *testing.T) {
	productId := fmt.Sprintf("m5-se-prop-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {}
function OnMessage(context) {
  context.SaveProperties({temperature: 1})
}
`
	port, cleanup := startMqtt5(t, productId, script)
	defer cleanup()

	c, cancel := mqtt5Connect(t, port, "dev-m5-ok")
	defer cancel()
	defer func() { _ = c.Disconnect(context.Background()) }()

	ctx, cancelPub := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPub()
	_, err := c.Publish(ctx, &paho.Publish{
		QoS: 0, Topic: "test", Payload: []byte(`{"temperature":1}`),
	})
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)
	// broker 仍可用：再发一条不崩
	_, err = c.Publish(ctx, &paho.Publish{
		QoS: 0, Topic: "test", Payload: []byte(`{"temperature":2}`),
	})
	require.NoError(t, err)
}

func TestMqtt5_ScriptError_ExplicitThrow(t *testing.T) {
	productId := fmt.Sprintf("m5-se-throw-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {
  throw "mqtt5 boom"
}
`
	port, cleanup := startMqtt5(t, productId, script)
	defer cleanup()

	c, cancel := mqtt5Connect(t, port, "dev-m5-ok")
	defer cancel()
	defer func() { _ = c.Disconnect(context.Background()) }()
	time.Sleep(300 * time.Millisecond)
	require.NotNil(t, core.GetSession("dev-m5-ok"))

	ctx, cancelPub := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPub()
	_, err := c.Publish(ctx, &paho.Publish{
		QoS: 0, Topic: "test",
		Payload: []byte(`{"deviceId":"dev-m5-ok","temperature":1}`),
	})
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	require.NotNil(t, core.GetSession("dev-m5-ok"))
}
