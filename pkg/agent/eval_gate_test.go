package agent

import (
	"encoding/json"
	"errors"
	"testing"

	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

// HITL 门闩基准：期望写在用例里，不是导出快照。
func TestEvalGates(t *testing.T) {
	type tc struct {
		name          string
		writeMode     string
		tool          string
		args          string
		hasPerms      bool
		product       bool
		network       string
		script        string
		userId        int64
		want          string
		wantTerminate bool
		reasonHas     string
	}
	cases := []tc{
		{name: "create_product_confirm", writeMode: "confirm", tool: ToolCreateProduct, args: `{"id":"HUMI01","name":"温湿度","networkType":"MQTT_BROKER"}`, hasPerms: true, want: ToolConfirm, wantTerminate: true},
		{name: "create_device_confirm", writeMode: "confirm", tool: ToolCreateDevice, args: `{"id":"1234","name":"1234设备","productId":"P1"}`, hasPerms: true, product: true, userId: 7, want: ToolConfirm, wantTerminate: true},
		{name: "create_product_auto", writeMode: "auto", tool: ToolCreateProduct, args: `{"id":"HUMI01","name":"温湿度","networkType":"MQTT_BROKER"}`, hasPerms: true, want: ToolAllow},
		{name: "create_product_no_perm", writeMode: "confirm", tool: ToolCreateProduct, args: `{"id":"HUMI01","name":"温湿度","networkType":"MQTT_BROKER"}`, want: ToolBlock, reasonHas: ReasonForbidden},
		{name: "create_product_bare_mqtt", writeMode: "confirm", tool: ToolCreateProduct, args: `{"id":"HUMI01","name":"温湿度","networkType":"MQTT"}`, hasPerms: true, want: ToolBlock, reasonHas: "网络类型"},
		{name: "save_tsl_dirty", writeMode: "confirm", tool: ToolSaveTSL, args: `{"productId":"P1","tsl":{"properties":[{"id":"t","name":"t","type":"float","specs":{}}]}}`, hasPerms: true, product: true, userId: 7, want: ToolBlock, reasonHas: "specs"},
		{name: "save_tsl_clean", writeMode: "confirm", tool: ToolSaveTSL, args: `{"productId":"P1","tsl":{"properties":[{"id":"temperature","name":"温度","type":"float","unit":"℃","scale":1}]}}`, hasPerms: true, product: true, userId: 7, want: ToolConfirm, wantTerminate: true},
		{name: "start_network_modbus", writeMode: "confirm", tool: ToolStartNetwork, args: `{"productId":"P1"}`, hasPerms: true, product: true, network: "MODBUS", script: "function OnInvoke(c){}", userId: 7, want: ToolBlock, reasonHas: "客户端"},
		{name: "save_product_config_confirm", writeMode: "confirm", tool: ToolSaveProductConfig, args: `{"productId":"P1","set":{"username":"u","password":"p"}}`, hasPerms: true, product: true, userId: 7, want: ToolConfirm, wantTerminate: true},
		{name: "run_script_is_read", writeMode: "confirm", tool: ToolRunScript, args: `{"productId":"P1","payload":"{}"}`, hasPerms: true, product: true, script: "function OnMessage(c){}", userId: 7, want: ToolAllow},
		{name: "get_debug_logs_is_read", writeMode: "confirm", tool: ToolGetDebugLogs, args: `{"productId":"P1"}`, hasPerms: true, product: true, userId: 7, want: ToolAllow},
	}

	orig := lookupProduct
	origDev := lookupDevice
	t.Cleanup(func() {
		lookupProduct = orig
		lookupDevice = origDev
	})
	lookupDevice = func(id string) (*models.DeviceModel, error) {
		return nil, errors.New("missing")
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			uid := c.userId
			if uid == 0 {
				uid = 1
			}
			lookupProduct = func(id string) (*models.ProductModel, error) {
				if !c.product {
					return nil, errors.New("product not exist")
				}
				nt := c.network
				if nt == "" {
					nt = "MQTT_BROKER"
				}
				return &models.ProductModel{Product: models.Product{
					Id: id, CreateId: uid, NetworkType: nt, Script: c.script,
				}}, nil
			}
			ctx := ToolContext{UserId: uid}
			if c.hasPerms {
				ctx.Perms = writeCtx(uid).Perms
			}
			g := BeforeToolCall(c.writeMode, ctx, c.tool, json.RawMessage(c.args))
			require.Equal(t, c.want, g.Action)
			require.Equal(t, c.wantTerminate, g.Terminate)
			if c.reasonHas != "" {
				require.Contains(t, g.Reason, c.reasonHas)
			}
		})
	}
}
