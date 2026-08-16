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
	ToolListProducts   = "list_products"
	ToolGetProduct     = "get_product"
	ToolGetTSLSchema   = "get_tsl_schema"
	ToolGetCodecDoc    = "get_codec_doc"
	ToolValidateTSL    = "validate_tsl"
	ToolValidateScript = "validate_script"
	ToolCreateProduct  = "create_product"
	ToolUpdateProduct  = "update_product"
	ToolSaveTSL        = "save_tsl"
	ToolSaveScript     = "save_script"
	ToolUpdateNetwork  = "update_network"
)

const (
	ToolAllow   = "allow"
	ToolConfirm = "confirm"
	ToolBlock   = "block"
)

// ToolContext is passed to every tool Validate/Execute.
type ToolContext struct {
	UserId int64
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
	registerTool(getTSLSchemaTool{})
	registerTool(getCodecDocTool{})
	registerTool(validateTSLTool{})
	registerTool(validateScriptTool{})
	registerTool(createProductTool{})
	registerTool(updateProductTool{})
	registerTool(saveTSLTool{})
	registerTool(saveScriptTool{})
	registerTool(updateNetworkTool{})
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

func HasDeployTool() bool {
	for _, t := range toolOrder {
		if t.Name() == "deploy_product" || t.Name() == "start_network" {
			return true
		}
	}
	return false
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

func BeforeToolCall(writeMode string, userId int64, name string, args json.RawMessage) ToolPreflight {
	if strings.TrimSpace(name) == "" {
		return ToolPreflight{Action: ToolBlock, Reason: "empty tool name", Terminate: true}
	}
	t, ok := LookupTool(name)
	if !ok {
		return ToolPreflight{Action: ToolBlock, Reason: "unknown tool " + name, Terminate: false}
	}
	ctx := ToolContext{UserId: userId}
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
	UserId int64
}

func (e ToolExec) Execute(name string, args json.RawMessage) (any, error) {
	t, ok := LookupTool(name)
	if !ok || t.Mutating() {
		return nil, fmt.Errorf("unknown or mutating tool %s", name)
	}
	ctx := ToolContext{UserId: e.UserId}
	if err := t.Validate(ctx, args); err != nil {
		return map[string]any{"ok": false, "message": err.Error()}, nil
	}
	out, _, err := t.Execute(ctx, args)
	return out, err
}

func ApplyMutating(userId int64, toolName string, payload string) (productId string, err error) {
	t, ok := LookupTool(toolName)
	if !ok || !t.Mutating() {
		return "", fmt.Errorf("unknown mutating tool")
	}
	args := json.RawMessage(payload)
	ctx := ToolContext{UserId: userId}
	if err := t.Validate(ctx, args); err != nil {
		return "", err
	}
	_, pid, err := t.Execute(ctx, args)
	return pid, err
}
