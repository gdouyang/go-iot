package codec

import (
	"net"
	"testing"

	"go-iot/pkg/option"

	"github.com/stretchr/testify/require"
)

func TestCheckScriptHTTPDisabled(t *testing.T) {
	prev := option.Global
	defer func() { option.Global = prev }()
	option.Global = &option.Options{
		Script: option.Script{HTTPEnabled: false, HTTPBlockPrivate: true},
	}
	err := checkScriptHTTPAllowed("https://example.com/x")
	require.Error(t, err)
	require.Contains(t, err.Error(), "disabled")
}

func TestCheckScriptHTTPBlockPrivate(t *testing.T) {
	prev := option.Global
	defer func() { option.Global = prev }()
	option.Global = &option.Options{
		Script: option.Script{HTTPEnabled: true, HTTPBlockPrivate: true},
	}

	cases := []string{
		"http://127.0.0.1/a",
		"http://localhost/a",
		"http://10.0.0.1/a",
		"http://192.168.1.1/a",
		"http://172.16.0.5/a",
		"http://[::1]/a",
	}
	for _, u := range cases {
		err := checkScriptHTTPAllowed(u)
		require.Error(t, err, u)
	}
}

func TestCheckScriptHTTPAllowPublicWhenBlocking(t *testing.T) {
	prev := option.Global
	defer func() { option.Global = prev }()
	option.Global = &option.Options{
		Script: option.Script{HTTPEnabled: true, HTTPBlockPrivate: true},
	}
	// 公网字面量 IP，不依赖 DNS
	err := checkScriptHTTPAllowed("https://1.1.1.1/cdn-cgi/trace")
	require.NoError(t, err)
}

func TestCheckScriptHTTPAllowPrivateWhenUnblocked(t *testing.T) {
	prev := option.Global
	defer func() { option.Global = prev }()
	option.Global = &option.Options{
		Script: option.Script{HTTPEnabled: true, HTTPBlockPrivate: false},
	}
	err := checkScriptHTTPAllowed("http://127.0.0.1/health")
	require.NoError(t, err)
}

func TestIsBlockedIP(t *testing.T) {
	require.True(t, isBlockedIP(net.ParseIP("127.0.0.1")))
	require.True(t, isBlockedIP(net.ParseIP("10.1.2.3")))
	require.True(t, isBlockedIP(net.ParseIP("192.168.0.1")))
	require.True(t, isBlockedIP(net.ParseIP("172.16.5.5")))
	require.False(t, isBlockedIP(net.ParseIP("8.8.8.8")))
}
