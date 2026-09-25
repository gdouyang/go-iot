package agent

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/codec"
	"go-iot/pkg/eventbus"
	productsvc "go-iot/pkg/service/product"
)

type getDebugLogsTool struct{}

func (getDebugLogsTool) Name() string   { return ToolGetDebugLogs }
func (getDebugLogsTool) Mutating() bool { return false }
func (getDebugLogsTool) Spec() client.CompatTool {
	return specOf(ToolGetDebugLogs, "监听产品调试事件总线指定时长(默认3000毫秒，最大15000毫秒)，获取最新输出的 console.log 调试日志(最多50条)。无历史记录，仅捕获监听期间到达的新日志。若无真实设备在线，推荐优先使用 run_script 试跑编解码脚本。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"},"deviceId":{"type":"string","description":"设备ID，选填，留空则监听该产品下所有设备"},"waitMs":{"type":"integer","description":"监听时长(毫秒)，默认 3000，最大 15000"},"limit":{"type":"integer","description":"最大返回条数，默认 50"},"level":{"type":"string","description":"日志级别过滤：debug|info|warn|error，留空不过滤"}},"required":["productId"]}`)
}
func (getDebugLogsTool) Validate(ctx ToolContext, args json.RawMessage) error {
	id, err := parseProductID(args)
	if err != nil {
		return err
	}
	return requireOwnedProduct(ctx.UserId, id)
}
func (getDebugLogsTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		ProductId string `json:"productId"`
		DeviceId  string `json:"deviceId"`
		WaitMs    int    `json:"waitMs"`
		Limit     int    `json:"limit"`
		Level     string `json:"level"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, "", err
	}
	id := strings.TrimSpace(p.ProductId)
	ob, err := lookupProduct(id)
	if err != nil {
		return nil, id, err
	}
	if err := productsvc.AssertOwner(ob, ctx.UserId); err != nil {
		return nil, id, err
	}
	wait := time.Duration(p.WaitMs) * time.Millisecond
	msgs := eventbus.CollectDebug(id, p.DeviceId, p.Limit, wait)
	level := strings.ToLower(strings.TrimSpace(p.Level))
	logs := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		if level != "" && level != "all" && m.Level != level {
			continue
		}
		logs = append(logs, map[string]any{
			"time": m.CreateTime, "level": m.Level, "deviceId": m.DeviceId, "data": m.Data,
		})
	}
	return map[string]any{
		"ok": true, "productId": id, "count": len(logs), "waitMs": wait.Milliseconds(),
		"logs": logs,
		"note": "only logs received while listening; empty if no device reported",
	}, id, nil
}

type runScriptTool struct{}

func (runScriptTool) Name() string   { return ToolRunScript }
func (runScriptTool) Mutating() bool { return false }
func (runScriptTool) Spec() client.CompatTool {
	return specOf(ToolRunScript, "在隔离沙箱中试跑编解码钩子函数。不会替换生产环境的运行时脚本，不写入时序数据库，也不会改变设备在线状态。默认钩子 OnMessage。若未传 script 则默认使用产品已保存的脚本。返回执行日志、属性上报(SaveProperties)、事件上报(SaveEvents)及下发报文等捕获结果。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"产品ID"},"hook":{"type":"string","description":"要试跑的钩子函数：OnMessage (默认)、OnConnect 或 OnInvoke"},"payload":{"description":"消息报文体：字符串或JSON对象"},"payloadHex":{"type":"string","description":"十六进制报文（如 010300000002c40b），适合二进制协议"},"topic":{"type":"string","description":"MQTT主题(仅MQTT协议需要)"},"deviceId":{"type":"string","description":"模拟的设备ID"},"script":{"type":"string","description":"自定义试跑脚本内容，选填；留空则使用产品当前保存的脚本"},"functionId":{"type":"string","description":"功能标识(仅 OnInvoke 需要)"},"data":{"type":"object","description":"功能调用入参对象(仅 OnInvoke 需要)"}},"required":["productId"]}`)
}
func (runScriptTool) Validate(ctx ToolContext, args json.RawMessage) error {
	p, err := parseRunScript(args)
	if err != nil {
		return err
	}
	if err := requireOwnedProduct(ctx.UserId, p.ProductId); err != nil {
		return err
	}
	if codec.NormalizeDryRunHook(p.Hook) == "" {
		return fmt.Errorf("hook must be OnMessage, OnConnect, or OnInvoke")
	}
	return nil
}
func (runScriptTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	p, err := parseRunScript(args)
	if err != nil {
		return nil, "", err
	}
	ob, err := lookupProduct(p.ProductId)
	if err != nil {
		return nil, p.ProductId, err
	}
	if err := productsvc.AssertOwner(ob, ctx.UserId); err != nil {
		return nil, p.ProductId, err
	}
	script := strings.TrimSpace(p.Script)
	if script == "" {
		script = ob.Script
	}
	if strings.TrimSpace(script) == "" {
		return map[string]any{"ok": false, "message": "产品没有配置编解码，请先 save_script 或传入 script"}, p.ProductId, nil
	}
	payload, err := decodeRunPayload(p)
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}, p.ProductId, nil
	}
	res := codec.DryRun(codec.DryRunOpts{
		ProductId:  p.ProductId,
		Script:     script,
		Hook:       p.Hook,
		Payload:    payload,
		Topic:      p.Topic,
		DeviceId:   p.DeviceId,
		FunctionId: p.FunctionId,
		Data:       p.Data,
	})
	out := map[string]any{
		"ok": res.Ok, "productId": p.ProductId, "hook": res.Hook,
		"logs": res.Logs, "savedProperties": res.SavedProperties, "savedEvents": res.SavedEvents,
		"deviceOnline": res.DeviceOnline, "sessionWrites": res.SessionWrites, "replies": res.Replies,
		"note": "isolated trial; did not persist properties or replace the live codec",
	}
	if res.FunctionMissing {
		out["functionMissing"] = true
	}
	if res.Error != "" {
		out["message"] = res.Error
	}
	return out, p.ProductId, nil
}

type runScriptArgs struct {
	ProductId  string          `json:"productId"`
	Hook       string          `json:"hook"`
	Payload    json.RawMessage `json:"payload"`
	PayloadHex string          `json:"payloadHex"`
	Topic      string          `json:"topic"`
	DeviceId   string          `json:"deviceId"`
	Script     string          `json:"script"`
	FunctionId string          `json:"functionId"`
	Data       map[string]any  `json:"data"`
}

func parseRunScript(args json.RawMessage) (runScriptArgs, error) {
	var p runScriptArgs
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return p, fmt.Errorf("invalid tool arguments")
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return p, err
	}
	if strings.TrimSpace(p.ProductId) == "" {
		return p, fmt.Errorf("productId is required")
	}
	return p, nil
}

func decodeRunPayload(p runScriptArgs) ([]byte, error) {
	if s := strings.TrimSpace(p.PayloadHex); s != "" {
		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("payloadHex: %w", err)
		}
		return b, nil
	}
	raw := bytesTrimSpace(p.Payload)
	if len(raw) == 0 {
		return nil, nil
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return []byte(s), nil
	}
	return raw, nil
}
