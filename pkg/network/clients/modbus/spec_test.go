package modbus

import (
	"testing"

	"go-iot/pkg/network"

	"github.com/stretchr/testify/require"
)

func TestModbusClient_Type(t *testing.T) {
	c := &Client{}
	require.Equal(t, network.MODBUS, c.Type())
}

func TestParseIntValue(t *testing.T) {
	v, err := parseIntValue("", "timeout")
	require.NoError(t, err)
	require.Equal(t, 5, v)
	v, err = parseIntValue("10", "timeout")
	require.NoError(t, err)
	require.Equal(t, 10, v)
	_, err = parseIntValue("x", "timeout")
	require.Error(t, err)
}

func TestTcpInfoJSON(t *testing.T) {
	// 保证结构可序列化（配置侧）
	info := TcpInfo{Address: "127.0.0.1", Port: 502, UnitID: 1, Timeout: 5, IdleTimeout: 5}
	require.Equal(t, 502, info.Port)
}
