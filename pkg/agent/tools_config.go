package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-iot/pkg/agent/client"
	productsvc "go-iot/pkg/service/product"
)

type getProductConfigTool struct{}

func (getProductConfigTool) Name() string   { return ToolGetProductConfig }
func (getProductConfigTool) Mutating() bool { return false }
func (getProductConfigTool) Spec() client.CompatTool {
	return specOf(ToolGetProductConfig, "获取产品级配置项（管理端产品详情'配置'页）。MQTT_BROKER 等网络类型的认证账号密码位于此处，而不是 update_network。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"}},"required":["productId"]}`)
}
func (getProductConfigTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, id)
}
func (getProductConfigTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
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
	items := productsvc.DisplayMetaconfig(ob.Metaconfig, ob.NetworkType)
	return map[string]any{"ok": true, "productId": id, "networkType": ob.NetworkType, "items": items}, id, nil
}

type saveProductConfigTool struct{}

func (saveProductConfigTool) Name() string   { return ToolSaveProductConfig }
func (saveProductConfigTool) Mutating() bool { return true }
func (saveProductConfigTool) Spec() client.CompatTool {
	return specOf(ToolSaveProductConfig, "更新产品级配置项（如 MQTT 认证账号密码或额外扩展配置）。采用合并更新机制，不会删除其他已有内置配置。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"},"set":{"type":"object","additionalProperties":{"type":"string"},"description":"要设置的键值对字典，例如 {\"username\":\"admin\",\"password\":\"123456\"}"}},"required":["productId","set"]}`)
}
func (saveProductConfigTool) Validate(ctx ToolContext, args json.RawMessage) error {
	p, err := parseProductConfigSet(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, p.ProductId)
}
func (saveProductConfigTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	p, err := parseProductConfigSet(args)
	if err != nil {
		return nil, "", err
	}
	items, err := productsvc.SaveMetaconfig(ctx.UserId, p.ProductId, p.Set)
	if err != nil {
		return nil, p.ProductId, err
	}
	return map[string]any{"ok": true, "productId": p.ProductId, "items": items}, p.ProductId, nil
}

type productConfigSet struct {
	ProductId string
	Set       map[string]string
}

func parseProductConfigSet(args json.RawMessage) (productConfigSet, error) {
	var raw struct {
		ProductId string          `json:"productId"`
		Set       json.RawMessage `json:"set"`
	}
	var out productConfigSet
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return out, fmt.Errorf("invalid tool arguments")
	}
	if err := json.Unmarshal(args, &raw); err != nil {
		return out, err
	}
	out.ProductId = strings.TrimSpace(raw.ProductId)
	if out.ProductId == "" {
		return out, fmt.Errorf("productId is required")
	}
	set, err := coerceStringMap(raw.Set)
	if err != nil {
		return out, err
	}
	if len(set) == 0 {
		return out, fmt.Errorf("set is required")
	}
	out.Set = set
	return out, nil
}

func coerceStringMap(raw json.RawMessage) (map[string]string, error) {
	if len(bytesTrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("set is required")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("set must be an object")
	}
	out := map[string]string{}
	for k, v := range m {
		k = strings.TrimSpace(k)
		if k == "" {
			return nil, fmt.Errorf("config key is empty")
		}
		if v == nil {
			out[k] = ""
			continue
		}
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}
