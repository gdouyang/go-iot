package tcpclient

import (
	"testing"

	"go-iot/pkg/network"

	"github.com/stretchr/testify/require"
)

func TestTcpClientSpec_FromJson(t *testing.T) {
	spec := &TcpClientSpec{}
	require.NoError(t, spec.FromJson(`{"host":"10.0.0.1","port":9000}`))
	require.Equal(t, "10.0.0.1", spec.Host)
	require.Equal(t, int32(9000), spec.Port)
}

func TestTcpClientSpec_FromNetwork(t *testing.T) {
	spec := &TcpClientSpec{}
	require.NoError(t, spec.FromNetwork(network.NetworkConf{
		Configuration: `{"host":"127.0.0.1","port":1}`,
	}))
	require.Equal(t, "127.0.0.1", spec.Host)
}

func TestTcpClient_Type(t *testing.T) {
	c := &TcpClient{}
	require.Equal(t, network.TCP_CLIENT, c.Type())
}
