package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/agent/codecdoc"
	"go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	devicemd "go-iot/pkg/models/device"
	productsvc "go-iot/pkg/service/product"
	"go-iot/pkg/tsl"
)

const (
	ToolListProducts     = "list_products"
	ToolGetProduct       = "get_product"
	ToolGetTSLSchema     = "get_tsl_schema"
	ToolGetCodecDoc      = "get_codec_doc"
	ToolValidateTSL      = "validate_tsl"
	ToolValidateScript   = "validate_script"
	ToolCreateProduct    = "create_product"
	ToolUpdateProduct    = "update_product"
	ToolSaveTSL          = "save_tsl"
	ToolSaveScript       = "save_script"
	ToolUpdateNetwork    = "update_network"
)

func ToolCatalog() []client.CompatTool {
	must := func(name, desc, schema string) client.CompatTool {
		return client.CompatTool{
			Type: "function",
			Function: client.FunctionSpec{
				Name:        name,
				Description: desc,
				Parameters:  json.RawMessage(schema),
			},
		}
	}
	return []client.CompatTool{
		must(ToolListProducts, "List current user's products", `{"type":"object","properties":{"keyword":{"type":"string","description":"按名称前缀匹配，可空（非模糊/infix）"},"limit":{"type":"integer"}},"additionalProperties":false}`),
		must(ToolGetProduct, "Get one product owned by the user", `{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		must(ToolGetTSLSchema, "Same TSL field contract as the system prompt. Optional; already injected each turn.", `{"type":"object","properties":{}}`),
		must(ToolGetCodecDoc, "Return the full codec markdown for a topic (complete API tables and samples, not a summary). topic is a networkType or split / cron. Omit topic to list docs. Call before save_script.", `{"type":"object","properties":{"topic":{"type":"string","description":"MQTT_BROKER / GOIOT_MQTT_BROKER / MQTT_CLIENT / TCP_SERVER / TCP_CLIENT / HTTP_SERVER / WEBSOCKET_SERVER / COAP_SERVER / MODBUS / split / cron，可空则返回目录"}},"additionalProperties":false}`),
		must(ToolValidateTSL, "Validate TSL JSON", `{"type":"object","properties":{"tsl":{"type":"object"}},"required":["tsl"]}`),
		must(ToolValidateScript, "Compile JS codec without registering", `{"type":"object","properties":{"script":{"type":"string"}},"required":["script"]}`),
		must(ToolCreateProduct, "Propose creating a product. Call as soon as id/name/networkType are known; do not ask the user to confirm in chat. The UI shows Apply/Reject and persists only after Apply. networkType must be an exact platform enum.", `{"type":"object","properties":{"id":{"type":"string","description":"产品ID，字母数字下划线中划线，最长32"},"name":{"type":"string"},"networkType":{"type":"string","enum":["MQTT_BROKER","GOIOT_MQTT_BROKER","TCP_SERVER","HTTP_SERVER","WEBSOCKET_SERVER","COAP_SERVER","MQTT_CLIENT","TCP_CLIENT","MODBUS"],"description":"必须是枚举原文字符串，禁止 MQTT/TCP 等简写"},"desc":{"type":"string"}},"required":["id","name","networkType"]}`),
		must(ToolUpdateProduct, "Update product name/desc", `{"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"},"desc":{"type":"string"}},"required":["id"]}`),
		must(ToolSaveTSL, "Save TSL for a product", `{"type":"object","properties":{"productId":{"type":"string"},"tsl":{"type":"object"}},"required":["productId","tsl"]}`),
		must(ToolSaveScript, "Save JS codec script", `{"type":"object","properties":{"productId":{"type":"string"},"script":{"type":"string"}},"required":["productId","script"]}`),
		must(ToolUpdateNetwork, "Update product network row (no new server ports)", `{"type":"object","properties":{"productId":{"type":"string"},"configuration":{"type":"object"}},"required":["productId"]}`),
	}
}

func HasDeployTool() bool {
	for _, t := range ToolCatalog() {
		if t.Function.Name == "deploy_product" || t.Function.Name == "start_network" {
			return true
		}
	}
	return false
}

func IsMutatingTool(name string) bool {
	switch name {
	case ToolCreateProduct, ToolUpdateProduct, ToolSaveTSL, ToolSaveScript, ToolUpdateNetwork:
		return true
	default:
		return false
	}
}

const (
	ToolAllow   = "allow"
	ToolConfirm = "confirm"
	ToolBlock   = "block"
)

// ToolPreflight is the beforeToolCall gate (pi-mono style).
// confirm: persist a draft and pause the run; the tool is not executed.
// block: write an error tool result and do not execute.
// allow: execute now.
type ToolPreflight struct {
	Action    string
	Reason    string
	Terminate bool
}

func BeforeToolCall(writeMode, name string, args json.RawMessage) ToolPreflight {
	if strings.TrimSpace(name) == "" {
		return ToolPreflight{Action: ToolBlock, Reason: "empty tool name", Terminate: true}
	}
	if !IsMutatingTool(name) {
		return ToolPreflight{Action: ToolAllow}
	}
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return ToolPreflight{Action: ToolBlock, Reason: "invalid tool arguments", Terminate: true}
	}
	if writeMode == "auto" {
		return ToolPreflight{Action: ToolAllow}
	}
	return ToolPreflight{Action: ToolConfirm, Terminate: true}
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

type ToolExec struct {
	UserId int64
}

func (e ToolExec) Execute(name string, args json.RawMessage) (any, error) {
	switch name {
	case ToolListProducts:
		return e.listProducts(args)
	case ToolGetProduct:
		return e.getProduct(args)
	case ToolGetTSLSchema:
		return tsl.Schema(), nil
	case ToolGetCodecDoc:
		return e.codecDoc(args)
	case ToolValidateTSL:
		return e.validateTSL(args)
	case ToolValidateScript:
		return e.validateScript(args)
	default:
		return nil, fmt.Errorf("unknown or mutating tool %s", name)
	}
}

func (e ToolExec) listProducts(args json.RawMessage) (any, error) {
	var p struct {
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 50 {
		p.Limit = 20
	}
	q := &models.PageQuery{PageNum: 1, PageSize: p.Limit}
	if strings.TrimSpace(p.Keyword) != "" {
		q.Condition = []core.SearchTerm{{Key: "name", Value: p.Keyword, Oper: core.LIKE}}
	}
	res, err := devicemd.PageProduct(q, e.UserId)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "total": res.TotalCount, "list": res.List}, nil
}

func (e ToolExec) getProduct(args json.RawMessage) (any, error) {
	var p struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	ob, err := devicemd.GetProductMust(p.Id)
	if err != nil {
		return nil, err
	}
	if ob.CreateId != e.UserId {
		return nil, productsvc.ErrNotOwner
	}
	return map[string]any{
		"ok": true, "id": ob.Id, "name": ob.Name, "networkType": ob.NetworkType,
		"state": ob.State, "storePolicy": ob.StorePolicy, "codecId": ob.CodecId,
	}, nil
}

func (e ToolExec) codecDoc(args json.RawMessage) (any, error) {
	var p struct {
		Topic string `json:"topic"`
	}
	_ = json.Unmarshal(args, &p)
	if strings.TrimSpace(p.Topic) == "" {
		return map[string]any{"ok": true, "topics": codecdoc.Topics()}, nil
	}
	md, err := codecdoc.Get(p.Topic)
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error(), "topics": codecdoc.Topics()}, nil
	}
	return map[string]any{"ok": true, "topic": p.Topic, "markdown": md}, nil
}

func (e ToolExec) validateTSL(args json.RawMessage) (any, error) {
	var p struct {
		TSL json.RawMessage `json:"tsl"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return map[string]any{"ok": false, "code": "invalid_args", "message": err.Error()}, nil
	}
	data := tsl.NewTslData()
	if err := data.FromJson(string(p.TSL)); err != nil {
		return map[string]any{"ok": false, "code": "invalid_tsl", "message": err.Error()}, nil
	}
	return map[string]any{"ok": true}, nil
}

func (e ToolExec) validateScript(args json.RawMessage) (any, error) {
	var p struct {
		Script string `json:"script"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return map[string]any{"ok": false, "code": "invalid_args"}, nil
	}
	if err := codec.CompileOnly(p.Script); err != nil {
		return map[string]any{"ok": false, "code": "compile_error", "message": err.Error()}, nil
	}
	return map[string]any{"ok": true, "bytes": len(p.Script)}, nil
}

func ApplyMutating(userId int64, toolName string, payload string) (productId string, err error) {
	switch toolName {
	case ToolCreateProduct:
		var p models.ProductModel
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return "", err
		}
		p.CreateId = userId
		return p.Id, productsvc.Add(&p)
	case ToolUpdateProduct:
		var p models.ProductModel
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return "", err
		}
		exist, err := devicemd.GetProductMust(p.Id)
		if err != nil {
			return p.Id, err
		}
		if err := productsvc.AssertOwner(exist, userId); err != nil {
			return p.Id, err
		}
		p.Metadata = ""
		p.Script = ""
		return p.Id, devicemd.UpdateProduct(&p)
	case ToolSaveTSL:
		var p struct {
			ProductId string          `json:"productId"`
			TSL       json.RawMessage `json:"tsl"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return "", err
		}
		return p.ProductId, productsvc.SaveTSL(userId, p.ProductId, string(p.TSL))
	case ToolSaveScript:
		var p struct {
			ProductId string `json:"productId"`
			Script    string `json:"script"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return "", err
		}
		return p.ProductId, productsvc.SaveScript(userId, p.ProductId, p.Script)
	case ToolUpdateNetwork:
		return "", fmt.Errorf("update_network apply is limited: only existing rows")
	default:
		return "", fmt.Errorf("unknown mutating tool")
	}
}
