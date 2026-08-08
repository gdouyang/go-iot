package coapserver_test

import (
	"fmt"
	"testing"

	"go-iot/pkg/network"
	coapserver "go-iot/pkg/network/servers/coap"
	"go-iot/pkg/network/testhelper"
)

// 验证：Start 后立即 Stop（server 赋值在 goroutine 内，可能 nil）
func TestCoapServer_ImmediateStop(t *testing.T) {
	port := testhelper.FreeUDPPort(t)
	productId := fmt.Sprintf("coap-stop-%d", port)
	testhelper.SetupProductDevice(t, productId, "coap-stop-0")
	conf := network.NetworkConf{
		ProductId:     productId,
		CodecId:       "script_codec",
		Port:          port,
		Script:        "function OnMessage(context) {}",
		Configuration: `{"host":"127.0.0.1","useTLS":false}`,
	}
	srv := coapserver.NewServer()
	if err := srv.Start(conf); err != nil {
		t.Fatal(err)
	}
	// Start 返回后立即 Stop（不 sleep）
	srv.Stop()
}
