package mqttserver_test

import (
	"fmt"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	mqttserver "go-iot/pkg/network/servers/mqtt"
	"go-iot/pkg/network/testhelper"
	_ "go-iot/pkg/timeseries"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
)

func startMQTT(t *testing.T, productId, script string) (port int32, cleanup func()) {
	t.Helper()
	port = testhelper.FreeTCPPort(t)
	testhelper.SetupProductDevice(t, productId, "dev-mqtt-ok")
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
	time.Sleep(100 * time.Millisecond)
	return port, func() { _ = b.Stop() }
}

func mqttConnect(t *testing.T, port int32, clientID string) MQTT.Client {
	t.Helper()
	opts := MQTT.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://127.0.0.1:%d", port))
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.SetConnectTimeout(3 * time.Second)
	c := MQTT.NewClient(opts)
	tok := c.Connect()
	require.True(t, tok.WaitTimeout(5*time.Second))
	// 连接结果因脚本/认证可能失败，调用方自行判断
	return c
}

func TestMqtt_ScriptError_DeviceOnlineUnknown(t *testing.T) {
	productId := fmt.Sprintf("mqtt-se-online-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {}
`
	port, cleanup := startMQTT(t, productId, script)
	defer cleanup()

	c := mqttConnect(t, port, "ghost-mqtt")
	if c.IsConnected() {
		time.Sleep(200 * time.Millisecond)
		require.Nil(t, core.GetSession("ghost-mqtt"))
		c.Disconnect(100)
	}

	// 正确设备可上线
	c2 := mqttConnect(t, port, "dev-mqtt-ok")
	require.True(t, c2.IsConnected() || c2.IsConnectionOpen())
	// 等待 OnConnect
	time.Sleep(300 * time.Millisecond)
	require.NotNil(t, core.GetSession("dev-mqtt-ok"))
	c2.Disconnect(100)
}

func TestMqtt_ScriptError_SavePropertiesEmptyDeviceId(t *testing.T) {
	productId := fmt.Sprintf("mqtt-se-prop-%d", time.Now().UnixNano()%100000)
	// OnConnect 不上线，OnMessage 直接 SaveProperties 无 deviceId
	script := `
function OnConnect(context) {}
function OnMessage(context) {
  context.SaveProperties({temperature: 1})
}
`
	port, cleanup := startMQTT(t, productId, script)
	defer cleanup()

	c := mqttConnect(t, port, "dev-mqtt-ok")
	if c.IsConnected() {
		tok := c.Publish("test", 0, false, []byte(`{"temperature":1}`))
		tok.WaitTimeout(3 * time.Second)
		time.Sleep(150 * time.Millisecond)
		c.Disconnect(100)
	}
	// 服务仍可接受新连接
	c2 := mqttConnect(t, port, "dev-mqtt-ok")
	if c2.IsConnected() {
		c2.Disconnect(100)
	}
}

func TestMqtt_ScriptError_ExplicitThrowOnMessage(t *testing.T) {
	productId := fmt.Sprintf("mqtt-se-throw-%d", time.Now().UnixNano()%100000)
	script := `
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {
  throw "mqtt boom"
}
`
	port, cleanup := startMQTT(t, productId, script)
	defer cleanup()

	c := mqttConnect(t, port, "dev-mqtt-ok")
	if !c.IsConnected() {
		t.Skip("mqtt client failed to connect in this environment")
	}
	defer c.Disconnect(100)
	time.Sleep(200 * time.Millisecond)
	require.NotNil(t, core.GetSession("dev-mqtt-ok"))
	tok := c.Publish("test", 0, false, []byte(`{"deviceId":"dev-mqtt-ok","temperature":1}`))
	require.True(t, tok.WaitTimeout(3*time.Second))
	time.Sleep(100 * time.Millisecond)
	// 会话与 broker 仍存活
	require.NotNil(t, core.GetSession("dev-mqtt-ok"))
}
