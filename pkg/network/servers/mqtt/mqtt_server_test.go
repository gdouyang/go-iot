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
	"go-iot/pkg/timeseries"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
)

const script = `
function OnConnect(context) {
	context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {
	var data = JSON.parse(context.MsgToString())
	if (data.name == 'f') {
		context.ReplyOk()
		return
	}
	context.SaveProperties(data)
}
function OnInvoke(context) {
	context.GetSession().Publish("test", JSON.stringify(context.GetMessage().Data))
}
`

func TestMqttBroker_OnlineAndPublish(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("mqtt-p-%d", port)
	deviceId := "dev-mqtt-1"
	testhelper.SetupProductDevice(t, productId, deviceId)

	conf := network.NetworkConf{
		Name:          "mqtt-it",
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false}`,
	}
	b := mqttserver.NewServer()
	require.NoError(t, b.Start(conf))
	t.Cleanup(func() { _ = b.Stop() })
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	opts := MQTT.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://127.0.0.1:%d", port))
	opts.SetClientID(deviceId)
	opts.SetCleanSession(true)
	client := MQTT.NewClient(opts)
	token := client.Connect()
	require.True(t, token.WaitTimeout(5*time.Second))
	require.NoError(t, token.Error())
	defer client.Disconnect(200)

	time.Sleep(200 * time.Millisecond)
	require.NotNil(t, core.GetSession(deviceId), "mqtt client should be online")

	before := timeseries.MockPropertiesCount()
	payload := fmt.Sprintf(`{"deviceId":"%s","temperature":12.1}`, deviceId)
	pub := client.Publish("test", 0, false, []byte(payload))
	require.True(t, pub.WaitTimeout(3*time.Second))
	require.NoError(t, pub.Error())
	time.Sleep(200 * time.Millisecond)

	require.Greater(t, timeseries.MockPropertiesCount(), before)
	saved := timeseries.MockLastProperties()
	require.NotNil(t, saved)
	require.Equal(t, deviceId, fmt.Sprint(saved["deviceId"]))
	require.EqualValues(t, 12.1, saved["temperature"])
}

func TestMqttBroker_UnknownDeviceRejected(t *testing.T) {
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("mqtt-p2-%d", port)
	testhelper.SetupProductDevice(t, productId, "known")

	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        script,
		Configuration: `{"host":"127.0.0.1","useTLS":false}`,
	}
	b := mqttserver.NewServer()
	require.NoError(t, b.Start(conf))
	t.Cleanup(func() { _ = b.Stop() })
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	opts := MQTT.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://127.0.0.1:%d", port))
	opts.SetClientID("ghost-mqtt")
	opts.SetConnectTimeout(3 * time.Second)
	client := MQTT.NewClient(opts)
	token := client.Connect()
	// 设备不存在时认证/上线失败，连接可能失败或连上后无 session
	ok := token.WaitTimeout(5 * time.Second)
	if ok && token.Error() == nil {
		time.Sleep(200 * time.Millisecond)
		require.Nil(t, core.GetSession("ghost-mqtt"))
		client.Disconnect(100)
	}
}

func TestMqttBroker_InvalidConfig(t *testing.T) {
	b := mqttserver.NewServer()
	err := b.Start(network.NetworkConf{
		ProductId:     "mqtt-bad",
		Port:          1,
		Configuration: `{bad`,
	})
	require.Error(t, err)
}
