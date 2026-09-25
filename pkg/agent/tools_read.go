package agent

import (
	"encoding/json"
	"strings"

	"go-iot/pkg/agent/client"
	productsvc "go-iot/pkg/service/product"
)

type getTSLTool struct{}

func (getTSLTool) Name() string   { return ToolGetTSL }
func (getTSLTool) Mutating() bool { return false }
func (getTSLTool) Spec() client.CompatTool {
	return specOf(ToolGetTSL, "获取指定产品当前保存的物模型(TSL) JSON。在修改已有物模型前必须先调用此工具获取基准。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"}},"required":["productId"]}`)
}
func (getTSLTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, id)
}
func (getTSLTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	id, err := parseProductID(args)
	if err != nil {
		return nil, "", err
	}
	ob, err := lookupProduct(id)
	if err != nil {
		return nil, id, err
	}
	if err := productsvc.AssertOwner(ob, ctx.UserId); err != nil {
		return nil, id, err
	}
	var tsl any = map[string]any{}
	if s := strings.TrimSpace(ob.Metadata); s != "" {
		if json.Unmarshal([]byte(s), &tsl) != nil {
			tsl = s
		}
	}
	return map[string]any{"ok": true, "productId": id, "tsl": tsl, "published": ob.State}, id, nil
}

type getScriptTool struct{}

func (getScriptTool) Name() string   { return ToolGetScript }
func (getScriptTool) Mutating() bool { return false }
func (getScriptTool) Spec() client.CompatTool {
	return specOf(ToolGetScript, "获取指定产品当前保存的编解码脚本源码。在修改已有脚本前应先调用此工具获取基准。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"}},"required":["productId"]}`)
}
func (getScriptTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, id)
}
func (getScriptTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	id, err := parseProductID(args)
	if err != nil {
		return nil, "", err
	}
	ob, err := lookupProduct(id)
	if err != nil {
		return nil, id, err
	}
	if err := productsvc.AssertOwner(ob, ctx.UserId); err != nil {
		return nil, id, err
	}
	return map[string]any{"ok": true, "productId": id, "script": ob.Script, "codecId": ob.CodecId}, id, nil
}

type getNetworkTool struct{}

func (getNetworkTool) Name() string   { return ToolGetNetwork }
func (getNetworkTool) Mutating() bool { return false }
func (getNetworkTool) Spec() client.CompatTool {
	return specOf(ToolGetNetwork, "获取产品绑定的网络配置信息（网络类型、监听端口、运行状态及配置参数，不包含证书敏感信息）。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"}},"required":["productId"]}`)
}
func (getNetworkTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, id)
}
func (getNetworkTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	id, err := parseProductID(args)
	if err != nil {
		return nil, "", err
	}
	nw, err := productsvc.GetNetwork(ctx.UserId, id)
	if err != nil {
		return nil, id, err
	}
	var cfg any
	if s := strings.TrimSpace(nw.Configuration); s != "" && json.Valid([]byte(s)) {
		_ = json.Unmarshal([]byte(s), &cfg)
	}
	return map[string]any{
		"ok": true, "productId": id, "name": nw.Name, "port": nw.Port,
		"type": nw.Type, "state": nw.State, "configuration": cfg,
	}, id, nil
}

type getCollectorTool struct{}

func (getCollectorTool) Name() string   { return ToolGetCollector }
func (getCollectorTool) Mutating() bool { return false }
func (getCollectorTool) Spec() client.CompatTool {
	return specOf(ToolGetCollector, "获取指定产品的 MODBUS 采集点表与采集组配置。若未配置则返回空对象。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"}},"required":["productId"]}`)
}
func (getCollectorTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, id)
}
func (getCollectorTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	id, err := parseProductID(args)
	if err != nil {
		return nil, "", err
	}
	raw, err := productsvc.GetCollector(ctx.UserId, id)
	if err != nil {
		return nil, id, err
	}
	var cfg any = map[string]any{}
	if len(raw) > 0 && json.Valid(raw) {
		_ = json.Unmarshal(raw, &cfg)
	}
	return map[string]any{"ok": true, "productId": id, "collector": cfg}, id, nil
}
