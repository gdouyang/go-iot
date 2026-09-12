package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/models"
	devicemd "go-iot/pkg/models/device"
	productsvc "go-iot/pkg/service/product"
)

const (
	ToolListProducts      = "list_products"
	ToolGetProduct        = "get_product"
	ToolGetProductConfig  = "get_product_config"
	ToolGetTSL            = "get_tsl"
	ToolGetScript         = "get_script"
	ToolGetNetwork        = "get_network"
	ToolGetCollector      = "get_collector"
	ToolGetTSLSchema      = "get_tsl_schema"
	ToolLoadSkill         = "load_skill"
	ToolValidateTSL       = "validate_tsl"
	ToolValidateScript    = "validate_script"
	ToolCreateProduct     = "create_product"
	ToolCreateDevice      = "create_device"
	ToolUpdateProduct     = "update_product"
	ToolSaveProductConfig = "save_product_config"
	ToolSaveTSL           = "save_tsl"
	ToolSaveScript        = "save_script"
	ToolUpdateNetwork     = "update_network"
	ToolSaveCollector     = "save_collector"
	ToolDeployProduct     = "deploy_product"
	ToolStartNetwork      = "start_network"
	ToolStopNetwork       = "stop_network"
	ToolGetDebugLogs      = "get_debug_logs"
	ToolRunScript         = "run_script"
	ToolGetDevice         = "get_device"
	ToolListDevices       = "list_devices"

	PermProductAdd  = "product-mgr:add"
	PermProductSave = "product-mgr:save"
	PermDeviceAdd   = "device-mgr:add"
)

const (
	ToolAllow   = "allow"
	ToolConfirm = "confirm"
	ToolBlock   = "block"
)

// ToolContext is passed to every tool Validate/Execute.
type ToolContext struct {
	UserId int64
	Perms  map[string]bool
}

func (ctx ToolContext) HasPerm(id string) bool {
	if ctx.Perms == nil {
		return false
	}
	return ctx.Perms[id]
}

func writePermFor(name string) string {
	switch name {
	case ToolCreateProduct:
		return PermProductAdd
	case ToolCreateDevice:
		return PermDeviceAdd
	default:
		return PermProductSave
	}
}

func requireWritePerm(ctx ToolContext, name string) error {
	need := writePermFor(name)
	if ctx.HasPerm(need) {
		return nil
	}
	return fmt.Errorf("%s: missing %s", ReasonForbidden, need)
}

// Tool is one catalog entry: schema, preflight, and apply/run live on the same type.
type Tool interface {
	Name() string
	Spec() client.CompatTool
	Mutating() bool
	Validate(ctx ToolContext, args json.RawMessage) error
	Execute(ctx ToolContext, args json.RawMessage) (result any, productId string, err error)
}

var (
	toolByName = map[string]Tool{}
	toolOrder  []Tool
)

func registerTool(t Tool) {
	toolByName[t.Name()] = t
	toolOrder = append(toolOrder, t)
}

func init() {
	registerTool(listProductsTool{})
	registerTool(getProductTool{})
	registerTool(getProductConfigTool{})
	registerTool(getTSLTool{})
	registerTool(getScriptTool{})
	registerTool(getNetworkTool{})
	registerTool(getCollectorTool{})
	registerTool(getTSLSchemaTool{})
	registerTool(loadSkillTool{})
	registerTool(validateTSLTool{})
	registerTool(validateScriptTool{})
	registerTool(createProductTool{})
	registerTool(createDeviceTool{})
	registerTool(updateProductTool{})
	registerTool(saveProductConfigTool{})
	registerTool(saveTSLTool{})
	registerTool(saveScriptTool{})
	registerTool(updateNetworkTool{})
	registerTool(saveCollectorTool{})
	registerTool(deployProductTool{})
	registerTool(startNetworkTool{})
	registerTool(stopNetworkTool{})
	registerTool(getDebugLogsTool{})
	registerTool(runScriptTool{})
	registerTool(getDeviceTool{})
	registerTool(listDevicesTool{})
}

func LookupTool(name string) (Tool, bool) {
	t, ok := toolByName[name]
	return t, ok
}

func ToolCatalog() []client.CompatTool {
	out := make([]client.CompatTool, 0, len(toolOrder))
	for _, t := range toolOrder {
		out = append(out, t.Spec())
	}
	return out
}

func ToolsCatalogMarkdown() string {
	var b strings.Builder
	b.WriteString("# 可用工具列表:\n")
	for _, t := range toolOrder {
		s := t.Spec()
		b.WriteString("- ")
		b.WriteString(s.Function.Name)
		b.WriteString(": ")
		b.WriteString(s.Function.Description)
		b.WriteByte('\n')
	}
	return b.String()
}

func HasDeployTool() bool {
	_, ok := LookupTool(ToolDeployProduct)
	return ok
}

func IsMutatingTool(name string) bool {
	t, ok := LookupTool(name)
	return ok && t.Mutating()
}

// ToolPreflight is the beforeToolCall gate (pi-mono style).
// confirm: persist a draft and pause the run; the tool is not executed.
// block: write an error tool result and do not execute.
// allow: execute now.
type ToolPreflight struct {
	Action    string
	Reason    string
	Terminate bool
}

// lookupProduct is overridden in tests. Default is ES GetProductMust.
var lookupProduct = func(id string) (*models.ProductModel, error) {
	return devicemd.GetProductMust(id)
}

func BeforeToolCall(writeMode string, ctx ToolContext, name string, args json.RawMessage) ToolPreflight {
	if strings.TrimSpace(name) == "" {
		return ToolPreflight{Action: ToolBlock, Reason: "empty tool name", Terminate: true}
	}
	t, ok := LookupTool(name)
	if !ok {
		return ToolPreflight{Action: ToolBlock, Reason: "unknown tool " + name, Terminate: false}
	}
	if t.Mutating() {
		if err := requireWritePerm(ctx, name); err != nil {
			return ToolPreflight{Action: ToolBlock, Reason: err.Error(), Terminate: false}
		}
	}
	if err := t.Validate(ctx, args); err != nil {
		return ToolPreflight{Action: ToolBlock, Reason: err.Error(), Terminate: false}
	}
	if !t.Mutating() {
		return ToolPreflight{Action: ToolAllow}
	}
	if writeMode == "auto" {
		return ToolPreflight{Action: ToolAllow}
	}
	return ToolPreflight{Action: ToolConfirm, Terminate: true}
}

func requireOwnedProduct(userId int64, productId string) error {
	id := strings.TrimSpace(productId)
	if id == "" {
		return fmt.Errorf("productId is required; create the product first")
	}
	exist, err := lookupProduct(id)
	if err != nil || exist == nil {
		return fmt.Errorf("product not found; call create_product first")
	}
	if err := productsvc.AssertOwner(exist, userId); err != nil {
		return err
	}
	return nil
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func specOf(name, desc, schema string) client.CompatTool {
	return client.CompatTool{
		Type: "function",
		Function: client.FunctionSpec{
			Name:        name,
			Description: desc,
			Parameters:  json.RawMessage(schema),
		},
	}
}

type ToolExec struct {
	Ctx ToolContext
}

func (e ToolExec) Execute(name string, args json.RawMessage) (any, error) {
	t, ok := LookupTool(name)
	if !ok || t.Mutating() {
		return nil, fmt.Errorf("unknown or mutating tool %s", name)
	}
	if err := t.Validate(e.Ctx, args); err != nil {
		return map[string]any{"ok": false, "message": err.Error()}, nil
	}
	out, _, err := t.Execute(e.Ctx, args)
	return out, err
}

func ApplyMutating(ctx ToolContext, toolName string, payload string) (productId string, err error) {
	t, ok := LookupTool(toolName)
	if !ok || !t.Mutating() {
		return "", fmt.Errorf("unknown mutating tool")
	}
	if err := requireWritePerm(ctx, toolName); err != nil {
		return "", err
	}
	args := json.RawMessage(payload)
	if err := t.Validate(ctx, args); err != nil {
		return "", err
	}
	_, pid, err := t.Execute(ctx, args)
	return pid, err
}
