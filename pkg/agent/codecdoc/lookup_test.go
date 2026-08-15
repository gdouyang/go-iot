package codecdoc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetKnownTopics(t *testing.T) {
	for _, topic := range Topics() {
		md, err := Get(topic)
		require.NoError(t, err, topic)
		require.NotEmpty(t, md, topic)
	}
	md, err := Get("splitFunc")
	require.NoError(t, err)
	require.Contains(t, md, "splitFunc")
}

func TestHTTPDocMatchesCode(t *testing.T) {
	md, err := Get("HTTP_SERVER")
	require.NoError(t, err)
	require.Contains(t, md, "OnStateChecker")
	require.Contains(t, md, "对接移动Onet平台")
	require.Contains(t, md, "启用/新增")
	require.Contains(t, md, "设备停用")
	require.Contains(t, md, "http://www.example.com")
	require.Contains(t, md, "HmacEncrypt")
	require.NotRegexp(t, `(?m)^\|\s*IsTextMessage\s*\|`, md)
	require.NotRegexp(t, `(?m)^\|\s*IsBinaryMessage\s*\|`, md)
	require.NotRegexp(t, `(?m)^\|\s*MsgToHexStr\s*\|`, md)
	require.Greater(t, len(md), 8000)
}

func TestGetUnknown(t *testing.T) {
	_, err := Get("MQTT")
	require.Error(t, err)
}

func TestMQTTBrokerDocMatchesJSFieldNames(t *testing.T) {
	md, err := Get("MQTT_BROKER")
	require.NoError(t, err)
	require.Contains(t, md, "FunctionId")
	require.Contains(t, md, "OnConnect")
	require.Contains(t, md, "GetPassword")
	require.Contains(t, md, "MessageID")
	require.Contains(t, md, "GetDeviceById(context.GetClientId())")
	require.Greater(t, len(md), 2000)
}

func TestTCPDocDoesNotInventSessionAPIs(t *testing.T) {
	for _, topic := range []string{"TCP_SERVER", "TCP_CLIENT"} {
		md, err := Get(topic)
		require.NoError(t, err, topic)
		require.NotContains(t, md, "session.GetDevice()")
		require.NotContains(t, md, "session.getId()")
		require.Contains(t, md, "context.GetDevice()")
		require.Contains(t, md, "session.GetDeviceId()")
		require.Contains(t, md, "new ReadProperty()")
		require.NotContains(t, md, "return new FireAlarm()}, function () {return 7}")
	}
}

func TestModbusDocHasSignedConverters(t *testing.T) {
	md, err := Get("MODBUS")
	require.NoError(t, err)
	require.Contains(t, md, "MsgToInt16")
	require.Contains(t, md, "MsgToInt32")
	require.Contains(t, md, "MsgToInt64")
	require.Contains(t, md, "boolean")
	require.True(t, strings.Contains(md, "MsgToBool"))
}

func TestMQTTClientHasPublishQos1(t *testing.T) {
	md, err := Get("MQTT_CLIENT")
	require.NoError(t, err)
	require.Contains(t, md, "PublishQos1")
	require.Contains(t, md, "MessageID")
}

func TestWebSocketSessionSendAPIs(t *testing.T) {
	md, err := Get("WEBSOCKET_SERVER")
	require.NoError(t, err)
	require.Contains(t, md, "SendText")
	require.Contains(t, md, "SendBinary")
	require.Contains(t, md, "IsTextMessage")
}

func TestCoAPDocMatchesCode(t *testing.T) {
	md, err := Get("COAP_SERVER")
	require.NoError(t, err)
	require.Contains(t, md, "DeviceOffline")
	require.Contains(t, md, "ResponseJSON")
	require.NotRegexp(t, `(?m)^\|\s*MsgToHexStr\s*\|`, md)
	require.NotRegexp(t, `(?m)^\|\s*GetHeader\s*\|`, md)
}
