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
	"go-iot/pkg/timeseries"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
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

func TestMqtt5Broker_OnlineAndPublish(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("mqtt5-p-%d", port)
	deviceId := "dev-mqtt5-1"
	testhelper.SetupProductDevice(t, productId, deviceId)

	conf := network.NetworkConf{
		Name:          "mqtt5-it",
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
	time.Sleep(150 * time.Millisecond)

	u, err := url.Parse(fmt.Sprintf("mqtt://127.0.0.1:%d", port))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: true,
		SessionExpiryInterval:         60,
		OnConnectError:                func(err error) { t.Logf("connect error: %v", err) },
		ClientConfig: paho.ClientConfig{
			ClientID: deviceId,
			OnClientError: func(err error) {
				t.Logf("client error: %v", err)
			},
		},
	}
	c, err := autopaho.NewConnection(ctx, cliCfg)
	require.NoError(t, err)
	require.NoError(t, c.AwaitConnection(ctx))

	time.Sleep(300 * time.Millisecond)
	require.NotNil(t, core.GetSession(deviceId), "mqtt5 device should be online")

	before := timeseries.MockPropertiesCount()
	payload := fmt.Sprintf(`{"deviceId":"%s","temperature":1.2}`, deviceId)
	_, err = c.Publish(ctx, &paho.Publish{
		QoS:     0,
		Topic:   "test",
		Payload: []byte(payload),
	})
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	require.Greater(t, timeseries.MockPropertiesCount(), before)
	saved := timeseries.MockLastProperties()
	require.NotNil(t, saved)
	require.Equal(t, deviceId, fmt.Sprint(saved["deviceId"]))
	require.EqualValues(t, 1.2, saved["temperature"])
	_ = c.Disconnect(ctx)
}

func TestMqtt5Broker_InvalidConfig(t *testing.T) {
	b := mqttserver.NewServer()
	err := b.Start(network.NetworkConf{
		ProductId:     "mqtt5-bad",
		Port:          1,
		Configuration: `{bad`,
	})
	require.Error(t, err)
}

// OnMessage 中 name=='f' 走 ReplyOk，不应 SaveProperties。
func TestMqtt5Broker_OnMessageReplyOkSkipsSaveProperties(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("mqtt5-reply-%d", port)
	deviceId := "dev-mqtt5-reply"
	testhelper.SetupProductDevice(t, productId, deviceId)

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
	time.Sleep(150 * time.Millisecond)

	u, err := url.Parse(fmt.Sprintf("mqtt://127.0.0.1:%d", port))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	c, err := autopaho.NewConnection(ctx, autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: true,
		ClientConfig:                  paho.ClientConfig{ClientID: deviceId},
	})
	require.NoError(t, err)
	require.NoError(t, c.AwaitConnection(ctx))
	time.Sleep(300 * time.Millisecond)
	require.NotNil(t, core.GetSession(deviceId))

	before := timeseries.MockPropertiesCount()
	_, err = c.Publish(ctx, &paho.Publish{
		QoS:     0,
		Topic:   "test",
		Payload: []byte(`{"name":"f"}`),
	})
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	require.Equal(t, before, timeseries.MockPropertiesCount(), "ReplyOk branch must not SaveProperties")
	_ = c.Disconnect(ctx)
}
