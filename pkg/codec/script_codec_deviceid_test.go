package codec_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/logger"
	"go-iot/pkg/option"

	"github.com/stretchr/testify/require"
)

func TestConsoleLogCarriesDeviceId(t *testing.T) {
	logger.InitNop()
	got := make(chan string, 1)
	pattern := "/device/p-console-dev/*/debug"
	handler := func(msg eventbus.Message) {
		if d, ok := msg.(*eventbus.DebugMessage); ok {
			got <- d.DeviceId
		}
	}
	eventbus.Subscribe(pattern, handler)
	t.Cleanup(func() { eventbus.UnSubscribe(pattern, handler) })

	script := `
function OnMessage(context) {
  console.log("hello")
}
`
	c, err := core.NewCodec(core.Script_Codec, "p-console-dev", script)
	require.NoError(t, err)
	err = c.OnMessage(&core.BaseContext{ProductId: "p-console-dev", DeviceId: "dev-9"})
	require.NoError(t, err)
	select {
	case id := <-got:
		require.Equal(t, "dev-9", id)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting debug log")
	}
}

func TestHttpRequestAsyncCompleteKeepsDeviceId(t *testing.T) {
	logger.InitNop()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	prev := option.Global
	t.Cleanup(func() { option.Global = prev })
	option.Global = &option.Options{
		Script: option.Script{HTTPEnabled: true, HTTPBlockPrivate: false},
	}

	got := make(chan string, 4)
	pattern := "/device/p-async-dev/*/debug"
	handler := func(msg eventbus.Message) {
		if d, ok := msg.(*eventbus.DebugMessage); ok && d.Data == "done" {
			got <- d.DeviceId
		}
	}
	eventbus.Subscribe(pattern, handler)
	t.Cleanup(func() { eventbus.UnSubscribe(pattern, handler) })

	script := `
function OnMessage(context) {
  globe.HttpRequestAsync({
    url: "` + srv.URL + `",
    method: "GET",
    complete: function(resp) {
      console.log("done")
    }
  })
}
`
	c, err := core.NewCodec(core.Script_Codec, "p-async-dev", script)
	require.NoError(t, err)
	err = c.OnMessage(&core.BaseContext{ProductId: "p-async-dev", DeviceId: "dev-async"})
	require.NoError(t, err)
	select {
	case id := <-got:
		require.Equal(t, "dev-async", id)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting async complete log")
	}
}
