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
	return specOf(ToolListProducts, "查询当前用户拥有的产品列表，支持按名称前缀匹配及分页条数限制。",
		`{"type":"object","properties":{"keyword":{"type":"string","description":"按名称前缀匹配，选填（非全局模糊匹配）"},"limit":{"type":"integer","description":"返回条数限制，默认 20，最大 50"}},"additionalProperties":false}`)
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
	return specOf(ToolGetProduct, "查询当前用户拥有的单个产品的基本元数据（名称、网络类型、发布状态、存储策略等）。",
		`{"type":"object","properties":{"id":{"type":"string","description":"产品ID"}},"required":["id"]}`)
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
		"hasTsl": strings.TrimSpace(ob.Metadata) != "", "hasScript": strings.TrimSpace(ob.Script) != "",
		"desc": ob.Desc,
	}, ob.Id, nil
}

type getTSLSchemaTool struct{}

func (getTSLSchemaTool) Name() string   { return ToolGetTSLSchema }
func (getTSLSchemaTool) Mutating() bool { return false }
func (getTSLSchemaTool) Spec() client.CompatTool {
	return specOf(ToolGetTSLSchema, "获取物模型(TSL) JSON Schema 字段契约规范。可选调用；系统每轮对话已默认注入核心规范。",
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
	return specOf(ToolLoadSkill, "加载特定网络类型或协议的编解码脚本规范文档(完整 Markdown API 手册，包含内置对象与示例)。name 为技能ID或网络类型枚举。留空 name 则返回技能目录。编写/保存脚本前强烈建议先调用。",
		`{"type":"object","properties":{"name":{"type":"string","description":"技能名称或网络类型枚举，留空则返回所有可用技能目录"}},"additionalProperties":false}`)
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
	return specOf(ToolValidateTSL, "校验物模型(TSL) JSON 格式及合法性，不写入存储。可用于提前发现物模型定义错误。",
		`{"type":"object","properties":{"tsl":{"type":"object","description":"物模型 JSON 对象，包含 properties、events、functions 等定义"}},"required":["tsl"]}`)
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
	return specOf(ToolValidateScript, "编译并语法检查 JavaScript 编解码脚本，不生效也不写入存储。可用于提前发现 JS 语法错误。",
		`{"type":"object","properties":{"script":{"type":"string","description":"待校验的 JavaScript 编解码脚本源码"}},"required":["script"]}`)
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
	return specOf(ToolCreateProduct, "提议创建新产品。一旦明确 id、name 和 networkType 即可直接调用，无需在聊天中反复向用户文字确认。前端将呈现应用/拒绝卡片由用户确认。networkType 必须是系统标准网络枚举。",
		`{"type":"object","properties":{"id":{"type":"string","description":"产品ID，仅支持字母、数字、下划线、中划线，最长32位"},"name":{"type":"string","description":"产品名称"},"networkType":{"type":"string","enum":["MQTT_BROKER","GOIOT_MQTT_BROKER","TCP_SERVER","HTTP_SERVER","WEBSOCKET_SERVER","COAP_SERVER","MQTT_CLIENT","TCP_CLIENT","MODBUS"],"description":"网络类型枚举字符串，必须全字匹配，禁止 MQTT/TCP 等简写"},"desc":{"type":"string","description":"产品描述，选填"}},"required":["id","name","networkType"]}`)
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
	return specOf(ToolUpdateProduct, "更新产品的基本信息（名称、描述）。不会修改物模型或编解码脚本。",
		`{"type":"object","properties":{"id":{"type":"string","description":"产品ID"},"name":{"type":"string","description":"产品新名称，选填"},"desc":{"type":"string","description":"产品新描述，选填"}},"required":["id"]}`)
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
	return specOf(ToolSaveTSL, "保存产品的物模型(TSL)定义。执行成功后草稿生效，需发布产品(deploy_product)后运行时才更新。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"},"tsl":{"type":"object","description":"物模型 JSON 对象，包含 properties、events、functions"}},"required":["productId","tsl"]}`)
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
	return specOf(ToolSaveScript, "保存产品的 JavaScript 编解码脚本。执行成功后草稿生效，需发布产品(deploy_product)后运行时才生效。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"},"script":{"type":"string","description":"JavaScript 编解码脚本源码"}},"required":["productId","script"]}`)
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
	return specOf(ToolUpdateNetwork, "更新产品绑定的网络配置参数（服务器主机/客户端连接选项等）。不会自动从端口池重新分配端口。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"},"name":{"type":"string","description":"网络配置名称，选填"},"configuration":{"type":"object","description":"网络配置 JSON 对象"},"port":{"type":"integer","description":"端口号，选填（直接更新现有记录端口，不自动分配新端口）"}},"required":["productId"]}`)
}
func (updateNetworkTool) Validate(ctx ToolContext, args json.RawMessage) error {
	p, err := parseNetworkPatch(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, p.ProductId)
}
func (updateNetworkTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	p, err := parseNetworkPatch(args)
	if err != nil {
		return nil, "", err
	}
	cfg := ""
	if len(bytesTrimSpace(p.Configuration)) > 0 {
		cfg = string(p.Configuration)
	}
	nw, err := productsvc.UpdateNetworkConfig(ctx.UserId, p.ProductId, p.Name, cfg, p.Port)
	if err != nil {
		return nil, p.ProductId, err
	}
	return map[string]any{"ok": true, "productId": p.ProductId, "port": nw.Port, "state": nw.State}, p.ProductId, nil
}

type networkPatch struct {
	ProductId     string          `json:"productId"`
	Name          string          `json:"name"`
	Configuration json.RawMessage `json:"configuration"`
	Port          int32           `json:"port"`
}

func parseNetworkPatch(args json.RawMessage) (networkPatch, error) {
	var p networkPatch
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return p, fmt.Errorf("invalid tool arguments")
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return p, err
	}
	if strings.TrimSpace(p.ProductId) == "" {
		return p, fmt.Errorf("productId is required")
	}
	if len(p.Configuration) > 0 && len(bytesTrimSpace(p.Configuration)) > 0 && !json.Valid(p.Configuration) {
		return p, fmt.Errorf("configuration must be JSON")
	}
	return p, nil
}

type saveCollectorTool struct{}

func (saveCollectorTool) Name() string   { return ToolSaveCollector }
func (saveCollectorTool) Mutating() bool { return true }
func (saveCollectorTool) Spec() client.CompatTool {
	return specOf(ToolSaveCollector, "保存 MODBUS 采集点表与采集组配置。必须在 save_tsl 之后调用以校验点表与属性的匹配性。仅适用于 MODBUS 网络类型。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"},"collector":{"type":"object","description":"采集配置 JSON 对象，包含 pointTables（点表）与 groups（采集组）"}},"required":["productId","collector"]}`)
}
func (saveCollectorTool) Validate(ctx ToolContext, args json.RawMessage) error {
	p, err := parseCollectorArgs(args)
	if err != nil {
		return err
	}
	if err := requireOwnedProduct(ctx.UserId, p.ProductId); err != nil {
		return err
	}
	ob, err := lookupProduct(p.ProductId)
	if err != nil {
		return err
	}
	return productsvc.ValidateCollector(ob.NetworkType, p.Collector, ob.Metadata)
}
func (saveCollectorTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	p, err := parseCollectorArgs(args)
	if err != nil {
		return nil, "", err
	}
	if err := productsvc.SaveCollector(ctx.UserId, p.ProductId, p.Collector); err != nil {
		return nil, p.ProductId, err
	}
	return map[string]any{"ok": true, "productId": p.ProductId}, p.ProductId, nil
}

type collectorArgs struct {
	ProductId string          `json:"productId"`
	Collector json.RawMessage `json:"collector"`
}

func parseCollectorArgs(args json.RawMessage) (collectorArgs, error) {
	var p collectorArgs
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return p, fmt.Errorf("invalid tool arguments")
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return p, err
	}
	if strings.TrimSpace(p.ProductId) == "" {
		return p, fmt.Errorf("productId is required")
	}
	if len(bytesTrimSpace(p.Collector)) == 0 {
		return p, fmt.Errorf("collector is required")
	}
	return p, nil
}

type deployProductTool struct{}

func (deployProductTool) Name() string   { return ToolDeployProduct }
func (deployProductTool) Mutating() bool { return true }
func (deployProductTool) Spec() client.CompatTool {
	return specOf(ToolDeployProduct, "发布产品，将草稿状态的物模型和编解码脚本编译并加载到运行时。发布要求物模型至少包含属性定义。MODBUS 在发布后采集器自动就绪；北向服务端(如 MQTT_BROKER、TCP_SERVER)仍需调用 start_network 启动网络监听。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"}},"required":["productId"]}`)
}
func (deployProductTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	if err := requireOwnedProduct(ctx.UserId, id); err != nil {
		return err
	}
	ob, err := lookupProduct(id)
	if err != nil {
		return err
	}
	_, err = productsvc.AssertDeployable(ob)
	return err
}
func (deployProductTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	id, err := parseProductID(args)
	if err != nil {
		return nil, "", err
	}
	if err := productsvc.Deploy(ctx.UserId, id); err != nil {
		return nil, id, err
	}
	return map[string]any{"ok": true, "productId": id, "published": true}, id, nil
}

type startNetworkTool struct{}

func (startNetworkTool) Name() string   { return ToolStartNetwork }
func (startNetworkTool) Mutating() bool { return true }
func (startNetworkTool) Spec() client.CompatTool {
	return specOf(ToolStartNetwork, "启动产品的北向网络服务端（如 MQTT_BROKER、TCP_SERVER、HTTP_SERVER、WEBSOCKET_SERVER、COAP_SERVER）。不适用于 MQTT_CLIENT、TCP_CLIENT 或 MODBUS 等南向类型（南向类型在发布或设备连接时自动激活）。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"}},"required":["productId"]}`)
}
func (startNetworkTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	if err := requireOwnedProduct(ctx.UserId, id); err != nil {
		return err
	}
	ob, err := lookupProduct(id)
	if err != nil {
		return err
	}
	return productsvc.AssertStartable(ob)
}
func (startNetworkTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	id, err := parseProductID(args)
	if err != nil {
		return nil, "", err
	}
	if err := productsvc.StartNetwork(ctx.UserId, id); err != nil {
		return nil, id, err
	}
	return map[string]any{"ok": true, "productId": id, "running": true}, id, nil
}

type stopNetworkTool struct{}

func (stopNetworkTool) Name() string   { return ToolStopNetwork }
func (stopNetworkTool) Mutating() bool { return true }
func (stopNetworkTool) Spec() client.CompatTool {
	return specOf(ToolStopNetwork, "停止产品的北向网络服务端，关闭端口监听。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"}},"required":["productId"]}`)
}
func (stopNetworkTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, id)
}
func (stopNetworkTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	id, err := parseProductID(args)
	if err != nil {
		return nil, "", err
	}
	if err := productsvc.StopNetwork(ctx.UserId, id); err != nil {
		return nil, id, err
	}
	return map[string]any{"ok": true, "productId": id, "running": false}, id, nil
}

func parseProductID(args json.RawMessage) (string, error) {
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return "", fmt.Errorf("invalid tool arguments")
	}
	var p struct {
		ProductId string `json:"productId"`
		Id        string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", err
	}
	id := strings.TrimSpace(p.ProductId)
	if id == "" {
		id = strings.TrimSpace(p.Id)
	}
	if id == "" {
		return "", fmt.Errorf("productId is required")
	}
	return id, nil
}
