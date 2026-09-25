package agent

import (
	"encoding/json"
	"strings"

	productsvc "go-iot/pkg/service/product"
)

const maxPreviewBytes = 24000

type previewDoc struct {
	Old       any    `json:"old"`
	New       any    `json:"new"`
	Truncated bool   `json:"truncated"`
	Warning   string `json:"warning,omitempty"`
	Published bool   `json:"published,omitempty"`
}

func BuildPreview(ctx ToolContext, toolName string, args json.RawMessage) (string, string) {
	pid := previewProductID(toolName, args)
	oldVal := previewOld(ctx, toolName, pid)
	newVal := previewNew(toolName, args)
	pub := false
	warn := ""
	if pid != "" {
		if ob, err := lookupProduct(pid); err == nil && ob != nil {
			pub = ob.State
		}
	}
	if pub && (toolName == ToolSaveTSL || toolName == ToolDeployProduct) {
		warn = "产品已发布：保存物模型可能变更时序表结构"
	}
	doc := previewDoc{Old: oldVal, New: newVal, Published: pub, Warning: warn}
	raw, truncated := marshalPreview(doc)
	if truncated {
		doc.Truncated = true
		raw, _ = marshalPreview(doc)
	}
	return string(raw), pid
}

func previewProductID(toolName string, args json.RawMessage) string {
	var p struct {
		ProductId string `json:"productId"`
		Id        string `json:"id"`
	}
	_ = json.Unmarshal(args, &p)
	switch toolName {
	case ToolCreateProduct, ToolUpdateProduct:
		return strings.TrimSpace(p.Id)
	default:
		if s := strings.TrimSpace(p.ProductId); s != "" {
			return s
		}
		return strings.TrimSpace(p.Id)
	}
}

func previewNew(toolName string, args json.RawMessage) any {
	var m map[string]any
	if json.Unmarshal(args, &m) != nil {
		return json.RawMessage(args)
	}
	switch toolName {
	case ToolCreateDevice:
		return map[string]any{"id": m["id"], "name": m["name"], "productId": m["productId"], "desc": m["desc"]}
	case ToolSaveTSL:
		return map[string]any{"productId": m["productId"], "tsl": m["tsl"]}
	case ToolSaveScript:
		return map[string]any{"productId": m["productId"], "script": m["script"]}
	case ToolSaveCollector:
		return map[string]any{"productId": m["productId"], "collector": m["collector"]}
	case ToolUpdateNetwork:
		return map[string]any{"productId": m["productId"], "name": m["name"], "port": m["port"], "configuration": m["configuration"]}
	case ToolSaveProductConfig:
		return map[string]any{"productId": m["productId"], "set": m["set"]}
	case ToolDeployProduct:
		return map[string]any{"productId": m["productId"], "published": true}
	case ToolStartNetwork:
		return map[string]any{"productId": m["productId"], "running": true}
	case ToolStopNetwork:
		return map[string]any{"productId": m["productId"], "running": false}
	default:
		return m
	}
}

func previewOld(ctx ToolContext, toolName, productId string) any {
	if productId == "" || toolName == ToolCreateProduct {
		return nil
	}
	ob, err := lookupProduct(productId)
	if err != nil || ob == nil {
		return nil
	}
	switch toolName {
	case ToolUpdateProduct:
		return map[string]any{"id": ob.Id, "name": ob.Name, "desc": ob.Desc}
	case ToolSaveProductConfig:
		items := productsvc.DisplayMetaconfig(ob.Metaconfig, ob.NetworkType)
		old := map[string]string{}
		for _, it := range items {
			old[it.Property] = it.Value
		}
		return map[string]any{"productId": productId, "config": old}
	case ToolSaveTSL:
		return map[string]any{"productId": productId, "tsl": parseJSONValue(ob.Metadata)}
	case ToolSaveScript:
		return map[string]any{"productId": productId, "script": ob.Script}
	case ToolSaveCollector:
		raw, err := productsvc.GetCollector(ctx.UserId, productId)
		if err != nil {
			return map[string]any{"productId": productId, "collector": nil}
		}
		return map[string]any{"productId": productId, "collector": parseJSONValue(string(raw))}
	case ToolUpdateNetwork:
		nw, err := productsvc.GetNetwork(ctx.UserId, productId)
		if err != nil || nw == nil {
			return nil
		}
		return map[string]any{
			"productId": productId, "name": nw.Name, "port": nw.Port,
			"type": nw.Type, "state": nw.State, "configuration": parseJSONValue(nw.Configuration),
		}
	case ToolDeployProduct:
		return map[string]any{"productId": productId, "published": ob.State}
	case ToolStartNetwork, ToolStopNetwork:
		running := false
		if nw, err := productsvc.GetNetwork(ctx.UserId, productId); err == nil && nw != nil {
			running = nw.State == "runing"
		}
		return map[string]any{"productId": productId, "running": running}
	default:
		return nil
	}
}

func parseJSONValue(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var v any
	if json.Unmarshal([]byte(s), &v) == nil {
		return v
	}
	return s
}

func marshalPreview(doc previewDoc) (json.RawMessage, bool) {
	raw, err := json.Marshal(doc)
	if err != nil {
		b, _ := json.Marshal(previewDoc{New: nil, Truncated: true})
		return b, true
	}
	if len(raw) <= maxPreviewBytes {
		return raw, false
	}
	neu, _ := doc.New.(map[string]any)
	if neu == nil {
		return raw[:maxPreviewBytes], true
	}
	slim := map[string]any{}
	for k, v := range neu {
		slim[k] = v
	}
	for _, key := range []string{"tsl", "script", "collector", "configuration"} {
		if slim[key] == nil {
			continue
		}
		slim[key] = map[string]any{"omitted": true, "hint": "full payload is applied on confirm"}
	}
	doc.New = slim
	doc.Truncated = true
	out, err := json.Marshal(doc)
	if err != nil {
		return raw[:maxPreviewBytes], true
	}
	return out, true
}
