package httpserver

import (
	"testing"

	"go-iot/pkg/network"

	"github.com/stretchr/testify/require"
)

func TestHttpServerSpec_FromJson(t *testing.T) {
	spec := &HttpServerSpec{}
	require.NoError(t, spec.FromJson(`{"host":"127.0.0.1","port":8080,"routers":[{"url":"/a"},{"url":""}]}`))
	require.Equal(t, "127.0.0.1", spec.Host)
	require.Len(t, spec.Routers, 1)
	require.Equal(t, "/a", spec.Routers[0].Url)
}

func TestHttpServerSpec_FromNetwork(t *testing.T) {
	spec := &HttpServerSpec{}
	require.NoError(t, spec.FromNetwork(network.NetworkConf{
		Configuration: `{"host":"0.0.0.0","useTLS":false}`,
	}))
	require.Equal(t, "0.0.0.0", spec.Host)
	require.False(t, spec.UseTLS)
}

func TestHttpServerSpec_InvalidJSON(t *testing.T) {
	spec := &HttpServerSpec{}
	require.Error(t, spec.FromJson(`{bad`))
}
