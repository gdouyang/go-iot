package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/agent/skills"
	"go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	devicemd "go-iot/pkg/models/device"
	"go-iot/pkg/network"
	productsvc "go-iot/pkg/service/product"
	"go-iot/pkg/tsl"
)

type listProductsTool struct{}

func (listProductsTool) Name() string   { return ToolListProducts }
func (listProductsTool) Mutating() bool { return false }
func (listProductsTool) Spec() client.CompatTool {
	return specOf(ToolListProducts, "List current user's products",
		`{"type":"object","properties":{"keyword":{"type":"string","description":"按名称前缀匹配，可空（非模糊/infix）"},"limit":{"type":"integer"}},"additionalProperties":false}`)
}
func (listProductsTool) Validate(ctx ToolContext, args json.RawMessage) error { return nil }
func (listProductsTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
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
	res, err := devicemd.PageProduct(q, ctx.UserId)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"ok": true, "total": res.TotalCount, "list": res.List}, "", nil
}

type getProductTool struct{}

func (getProductTool) Name() string   { return ToolGetProduct }
func (getProductTool) Mutating() bool { return false }
func (getProductTool) Spec() client.CompatTool {
	return specOf(ToolGetProduct, "Get one product owned by the user",
		`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`)
}
func (getProductTool) Validate(ctx ToolContext, args json.RawMessage) error {
	var p struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.Id) == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}
func (getProductTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, "", err
	}
	ob, err := lookupProduct(p.Id)
	if err != nil {
		return nil, "", err
	}
	if err := productsvc.AssertOwner(ob, ctx.UserId); err != nil {
		return nil, "", err
	}
	return map[string]any{
		"ok": true, "id": ob.Id, "name": ob.Name, "networkType": ob.NetworkType,
		"state": ob.State, "storePolicy": ob.StorePolicy, "codecId": ob.CodecId,
	}, ob.Id, nil
}

type getTSLSchemaTool struct{}

func (getTSLSchemaTool) Name() string   { return ToolGetTSLSchema }
func (getTSLSchemaTool) Mutating() bool { return false }
func (getTSLSchemaTool) Spec() client.CompatTool {
	return specOf(ToolGetTSLSchema, "Same TSL field contract as the system prompt. Optional; already injected each turn.",
		`{"type":"object","properties":{}}`)
}
func (getTSLSchemaTool) Validate(ctx ToolContext, args json.RawMessage) error { return nil }
func (getTSLSchemaTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	return tsl.Schema(), "", nil
}

type loadSkillTool struct{}

func (loadSkillTool) Name() string   { return ToolLoadSkill }
func (loadSkillTool) Mutating() bool { return false }
func (loadSkillTool) Spec() client.CompatTool {
	return specOf(ToolLoadSkill, "Load a codec skill (full API markdown, not a summary). name is a skill id or networkType. Omit name to list skills. Call before save_script.",
		`{"type":"object","properties":{"name":{"type":"string","description":"skill 名或网络类型枚举，可空则返回目录"}},"additionalProperties":false}`)
}
func (loadSkillTool) Validate(ctx ToolContext, args json.RawMessage) error { return nil }
func (loadSkillTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(args, &p)
	if strings.TrimSpace(p.Name) == "" {
		return map[string]any{"ok": true, "skills": skills.Summaries()}, "", nil
	}
	s, md, err := skills.Get(p.Name)
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error(), "skills": skills.Summaries()}, "", nil
	}
	return map[string]any{"ok": true, "name": s.Name, "markdown": md}, "", nil
}

type validateTSLTool struct{}

func (validateTSLTool) Name() string   { return ToolValidateTSL }
func (validateTSLTool) Mutating() bool { return false }
func (validateTSLTool) Spec() client.CompatTool {
	return specOf(ToolValidateTSL, "Validate TSL JSON",
		`{"type":"object","properties":{"tsl":{"type":"object"}},"required":["tsl"]}`)
}
func (validateTSLTool) Validate(ctx ToolContext, args json.RawMessage) error {
	var p struct {
		TSL json.RawMessage `json:"tsl"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	if len(bytesTrimSpace(p.TSL)) == 0 {
		return fmt.Errorf("tsl is required")
	}
	return nil
}
func (validateTSLTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		TSL json.RawMessage `json:"tsl"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return map[string]any{"ok": false, "code": "invalid_args", "message": err.Error()}, "", nil
	}
	if err := tsl.NewTslData().FromJson(string(p.TSL)); err != nil {
		return map[string]any{"ok": false, "code": "invalid_tsl", "message": err.Error()}, "", nil
	}
	return map[string]any{"ok": true}, "", nil
}

type validateScriptTool struct{}

func (validateScriptTool) Name() string   { return ToolValidateScript }
func (validateScriptTool) Mutating() bool { return false }
func (validateScriptTool) Spec() client.CompatTool {
	return specOf(ToolValidateScript, "Compile JS codec without registering",
		`{"type":"object","properties":{"script":{"type":"string"}},"required":["script"]}`)
}
func (validateScriptTool) Validate(ctx ToolContext, args json.RawMessage) error {
	var p struct {
		Script string `json:"script"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.Script) == "" {
		return fmt.Errorf("script is required")
	}
	return nil
}
func (validateScriptTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		Script string `json:"script"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return map[string]any{"ok": false, "code": "invalid_args"}, "", nil
	}
	if err := codec.CompileOnly(p.Script); err != nil {
		return map[string]any{"ok": false, "code": "compile_error", "message": err.Error()}, "", nil
	}
	return map[string]any{"ok": true, "bytes": len(p.Script)}, "", nil
}

type createProductTool struct{}

func (createProductTool) Name() string   { return ToolCreateProduct }
func (createProductTool) Mutating() bool { return true }
func (createProductTool) Spec() client.CompatTool {
	return specOf(ToolCreateProduct, "Propose creating a product. Call as soon as id/name/networkType are known; do not ask the user to confirm in chat. The UI shows Apply/Reject and persists only after Apply. networkType must be an exact platform enum.",
		`{"type":"object","properties":{"id":{"type":"string","description":"产品ID，字母数字下划线中划线，最长32"},"name":{"type":"string"},"networkType":{"type":"string","enum":["MQTT_BROKER","GOIOT_MQTT_BROKER","TCP_SERVER","HTTP_SERVER","WEBSOCKET_SERVER","COAP_SERVER","MQTT_CLIENT","TCP_CLIENT","MODBUS"],"description":"必须是枚举原文字符串，禁止 MQTT/TCP 等简写"},"desc":{"type":"string"}},"required":["id","name","networkType"]}`)
}
func (createProductTool) Validate(ctx ToolContext, args json.RawMessage) error {
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return fmt.Errorf("invalid tool arguments")
	}
	var p models.ProductModel
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	p.Id = devicemd.NormalizeId(p.Id)
	if p.Id == "" || strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("id and name must be present")
	}
	if len(p.Id) > 32 || !devicemd.DeviceIdValid(p.Id) {
		return fmt.Errorf("产品ID格式错误")
	}
	if !network.IsValidNetType(p.NetworkType) {
		return fmt.Errorf("不支持的网络类型: %s", p.NetworkType)
	}
	if exist, err := lookupProduct(p.Id); err == nil && exist != nil {
		return fmt.Errorf("产品[%s]已存在，不要再 create_product", p.Id)
	}
	return nil
}
func (createProductTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p models.ProductModel
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, "", err
	}
	p.CreateId = ctx.UserId
	if err := productsvc.Add(&p); err != nil {
		return nil, p.Id, err
	}
	return map[string]any{"ok": true, "productId": p.Id}, p.Id, nil
}

type updateProductTool struct{}

func (updateProductTool) Name() string   { return ToolUpdateProduct }
func (updateProductTool) Mutating() bool { return true }
func (updateProductTool) Spec() client.CompatTool {
	return specOf(ToolUpdateProduct, "Update product name/desc",
		`{"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"},"desc":{"type":"string"}},"required":["id"]}`)
}
func (updateProductTool) Validate(ctx ToolContext, args json.RawMessage) error {
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return fmt.Errorf("invalid tool arguments")
	}
	var p struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, p.Id)
}
func (updateProductTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p models.ProductModel
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, "", err
	}
	exist, err := lookupProduct(p.Id)
	if err != nil {
		return nil, p.Id, err
	}
	if err := productsvc.AssertOwner(exist, ctx.UserId); err != nil {
		return nil, p.Id, err
	}
	p.Metadata = ""
	p.Script = ""
	if err := devicemd.UpdateProduct(&p); err != nil {
		return nil, p.Id, err
	}
	return map[string]any{"ok": true, "productId": p.Id}, p.Id, nil
}

type saveTSLTool struct{}

func (saveTSLTool) Name() string   { return ToolSaveTSL }
func (saveTSLTool) Mutating() bool { return true }
func (saveTSLTool) Spec() client.CompatTool {
	return specOf(ToolSaveTSL, "Save TSL for a product",
		`{"type":"object","properties":{"productId":{"type":"string"},"tsl":{"type":"object"}},"required":["productId","tsl"]}`)
}
func (saveTSLTool) Validate(ctx ToolContext, args json.RawMessage) error {
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return fmt.Errorf("invalid tool arguments")
	}
	var p struct {
		ProductId string          `json:"productId"`
		TSL       json.RawMessage `json:"tsl"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	if err := requireOwnedProduct(ctx.UserId, p.ProductId); err != nil {
		return err
	}
	if len(bytesTrimSpace(p.TSL)) == 0 {
		return fmt.Errorf("tsl is required")
	}
	return tsl.NewTslData().FromJson(string(p.TSL))
}
func (saveTSLTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		ProductId string          `json:"productId"`
		TSL       json.RawMessage `json:"tsl"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, "", err
	}
	if err := productsvc.SaveTSL(ctx.UserId, p.ProductId, string(p.TSL)); err != nil {
		return nil, p.ProductId, err
	}
	return map[string]any{"ok": true, "productId": p.ProductId}, p.ProductId, nil
}

type saveScriptTool struct{}

func (saveScriptTool) Name() string   { return ToolSaveScript }
func (saveScriptTool) Mutating() bool { return true }
func (saveScriptTool) Spec() client.CompatTool {
	return specOf(ToolSaveScript, "Save JS codec script",
		`{"type":"object","properties":{"productId":{"type":"string"},"script":{"type":"string"}},"required":["productId","script"]}`)
}
func (saveScriptTool) Validate(ctx ToolContext, args json.RawMessage) error {
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return fmt.Errorf("invalid tool arguments")
	}
	var p struct {
		ProductId string `json:"productId"`
		Script    string `json:"script"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	if err := requireOwnedProduct(ctx.UserId, p.ProductId); err != nil {
		return err
	}
	if strings.TrimSpace(p.Script) == "" {
		return fmt.Errorf("script is required")
	}
	return codec.CompileOnly(p.Script)
}
func (saveScriptTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		ProductId string `json:"productId"`
		Script    string `json:"script"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, "", err
	}
	if err := productsvc.SaveScript(ctx.UserId, p.ProductId, p.Script); err != nil {
		return nil, p.ProductId, err
	}
	return map[string]any{"ok": true, "productId": p.ProductId}, p.ProductId, nil
}

type updateNetworkTool struct{}

func (updateNetworkTool) Name() string   { return ToolUpdateNetwork }
func (updateNetworkTool) Mutating() bool { return true }
func (updateNetworkTool) Spec() client.CompatTool {
	return specOf(ToolUpdateNetwork, "Update product network row (no new server ports)",
		`{"type":"object","properties":{"productId":{"type":"string"},"configuration":{"type":"object"}},"required":["productId"]}`)
}
func (updateNetworkTool) Validate(ctx ToolContext, args json.RawMessage) error {
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return fmt.Errorf("invalid tool arguments")
	}
	var p struct {
		ProductId string `json:"productId"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, p.ProductId)
}
func (updateNetworkTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		ProductId string `json:"productId"`
	}
	_ = json.Unmarshal(args, &p)
	return nil, p.ProductId, fmt.Errorf("update_network apply is limited: only existing rows")
}
