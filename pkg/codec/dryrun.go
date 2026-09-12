// Package codec 提供物联网设备报文编解码的脚本运行时及辅助工具。
// dryrun.go 实现了编解码脚本的安全试跑（Dry Run / 演练）沙箱引擎。
//
// 核心设计与安全隔离机制：
// 1. 独立运行：使用临时独立的 Goja JavaScript 虚拟机池，不替换或影响线上已加载的生产 Codec。
// 2. 存储隔离：拦截 SaveProperties、SaveEvents 等操作，解析结果仅暂存于内存，不写入 Elasticsearch / TDengine，不推入 EventBus 业务流。
// 3. 网络隔离：使用 Mock 的 dryRunSession，拦截 Send、Publish 等下行指令，绝不向真实物理设备或网络套接字发送报文。
// 4. 状态隔离：拦截 DeviceOnline，不向全局 Redis / 内存注册真实设备会话。
// 5. 超时与崩溃防护：默认 5 秒超时保护（通过 vm.Interrupt 防止死循环），并通过 recover 捕获脚本运行时 panic。
//
// 典型应用场景：
// - 管理端 AI 助手（Agent）在为产品生成或调整编解码脚本时，通过 run_script 工具在沙箱中演练验证；
// - 单元测试或离线调试脚本解析逻辑。
package codec

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	logs "go-iot/pkg/logger"

	"github.com/dop251/goja"
)

// dryRunTimeout 单次试跑的默认超时时间（防止脚本内死循环 while(true) 等）
const dryRunTimeout = 5 * time.Second

// DryRunOpts 定义编解码脚本试跑的入参选项
type DryRunOpts struct {
	ProductId  string         // 产品 ID（缺省时为 "dry-run"）
	Script     string         // 待试跑的 JavaScript 编解码脚本源码
	Hook       string         // 待触发的钩子函数：OnMessage、OnConnect、OnInvoke
	Payload    []byte         // 模拟接收到的设备上行原始报文（Hook 为 OnMessage 时有效）
	Topic      string         // 模拟的 MQTT Topic
	DeviceId   string         // 模拟的设备 ID（缺省时为 "dry-run"）
	ClientId   string         // 模拟的客户端 Client ID（MQTT 连接等场景）
	Username   string         // 模拟的认证用户名
	Password   string         // 模拟的认证密码
	FunctionId string         // 调用的功能 ID（Hook 为 OnInvoke 时有效）
	Data       map[string]any // 功能调用参数（Hook 为 OnInvoke 时有效）
	Timeout    time.Duration  // 自定义超时时间（<=0 时使用默认 dryRunTimeout）
}

// DryRunWrite 记录脚本尝试向设备会话下发指令的动作（拦截 Send / Publish）
type DryRunWrite struct {
	Method string `json:"method"`          // 调用的方法名（如 Send, SendHex, Publish 等）
	Topic  string `json:"topic,omitempty"` // 发布的目标 Topic（Publish 时有效）
	Data   string `json:"data"`            // 下发的实际数据内容
}

// DryRunEvent 记录脚本触发的物模型事件
type DryRunEvent struct {
	EventId string `json:"eventId"` // 物模型事件 ID
	Data    any    `json:"data"`    // 事件上报的数据负载
}

// DryRunResult 记录脚本试跑执行后的完整输出与诊断信息
type DryRunResult struct {
	Ok              bool             `json:"ok"`                        // 脚本是否成功执行完成
	Hook            string           `json:"hook"`                      // 实际执行的钩子名称
	FunctionMissing bool             `json:"functionMissing,omitempty"` // 脚本中是否缺少对应的钩子函数实现
	Error           string           `json:"error,omitempty"`           // 编译错误、执行异常或超时错误信息
	Logs            []map[string]any `json:"logs"`                      // 试跑期间 console.log 输出的调试日志
	SavedProperties []map[string]any `json:"savedProperties"`           // 脚本通过 SaveProperties 解析出的物模型属性
	SavedEvents     []DryRunEvent    `json:"savedEvents"`               // 脚本触发保存的事件列表
	DeviceOnline    []string         `json:"deviceOnline"`              // 脚本尝试标记上线的设备 ID 列表
	SessionWrites   []DryRunWrite    `json:"sessionWrites"`             // 脚本尝试向设备下发的控制报文
	Replies         []string         `json:"replies"`                   // 调用的应答结果（如 ok, fail, authFail）
}

// dryRunSession 模拟的虚拟设备会话，将所有向设备写入的操作记录到切片中，不进行真实网络 I/O
type dryRunSession struct {
	deviceId string
	writes   *[]DryRunWrite
}

func (s *dryRunSession) Disconnect() error { return nil }
func (s *dryRunSession) Close() error      { return nil }
func (s *dryRunSession) GetDeviceId() string {
	return s.deviceId
}
func (s *dryRunSession) SetDeviceId(id string) { s.deviceId = id }
func (s *dryRunSession) GetConInfo() map[string]any {
	return map[string]any{"dryRun": true}
}
func (s *dryRunSession) append(method, topic, data string) {
	*s.writes = append(*s.writes, DryRunWrite{Method: method, Topic: topic, Data: data})
}
func (s *dryRunSession) Send(msg string) error {
	s.append("Send", "", msg)
	return nil
}
func (s *dryRunSession) SendHex(msgHex string) error {
	s.append("SendHex", "", msgHex)
	return nil
}
func (s *dryRunSession) SendText(msg string) error {
	s.append("SendText", "", msg)
	return nil
}
func (s *dryRunSession) SendBinary(msg string) error {
	s.append("SendBinary", "", msg)
	return nil
}
func (s *dryRunSession) Publish(topic, data string) error {
	s.append("Publish", topic, data)
	return nil
}
func (s *dryRunSession) PublishHex(topic, data string) error {
	s.append("PublishHex", topic, data)
	return nil
}

// DryRunContext 是专用于试跑演练的 MessageContext 实现。
// 它截获所有与设备上下线、属性入库、事件上报、会话写入相关的调用，
// 仅在内存中收集操作结果，绝不触碰底层的真实设备会话、Redis 注册表与 Elasticsearch/TDengine 时序库。
type DryRunContext struct {
	core.BaseContext
	payload     []byte           // 模拟接收到的报文内容
	topic       string           // 模拟的 Topic
	clientId    string           // 模拟的 ClientId
	username    string           // 模拟的用户名
	password    string           // 模拟的密码
	invoke      *core.FuncInvoke // 功能调用的参数（OnInvoke 时）
	savedProps  []map[string]any // 收集到的解析属性
	savedEvents []DryRunEvent    // 收集到的事件
	online      []string         // 收集到的上线设备 ID
	replies     []string         // 收集到的应答内容
	sess        *dryRunSession   // 关联的 Mock 虚拟会话
}

func (ctx *DryRunContext) GetMessage() any {
	if ctx.invoke != nil {
		return *ctx.invoke
	}
	return ctx.payload
}
func (ctx *DryRunContext) MsgToString() string { return string(ctx.payload) }
func (ctx *DryRunContext) MsgToHexStr() string { return hex.EncodeToString(ctx.payload) }
func (ctx *DryRunContext) Topic() string       { return ctx.topic }
func (ctx *DryRunContext) MessageID() uint16   { return 0 }
func (ctx *DryRunContext) GetClientId() string {
	if ctx.clientId != "" {
		return ctx.clientId
	}
	return ctx.DeviceId
}
func (ctx *DryRunContext) GetUserName() string { return ctx.username }
func (ctx *DryRunContext) GetPassword() string { return ctx.password }
func (ctx *DryRunContext) GetSession() core.Session {
	return ctx.sess
}
func (ctx *DryRunContext) AuthFail() {
	ctx.replies = append(ctx.replies, "authFail")
}
func (ctx *DryRunContext) DeviceOnline(deviceId string) error {
	deviceId = strings.TrimSpace(deviceId)
	if deviceId == "" {
		return nil
	}
	ctx.online = append(ctx.online, deviceId)
	ctx.DeviceId = deviceId
	if ctx.sess != nil {
		ctx.sess.SetDeviceId(deviceId)
	}
	return nil
}
func (ctx *DryRunContext) KeepAlive(string) {}
func (ctx *DryRunContext) SaveProperties(data map[string]any) error {
	ctx.savedProps = append(ctx.savedProps, cloneMap(data))
	return nil
}
func (ctx *DryRunContext) SaveEvents(eventId string, data any) error {
	ctx.savedEvents = append(ctx.savedEvents, DryRunEvent{EventId: eventId, Data: data})
	return nil
}
func (ctx *DryRunContext) ReplyOk() {
	ctx.replies = append(ctx.replies, "ok")
}
func (ctx *DryRunContext) ReplyFail(resp string) {
	ctx.replies = append(ctx.replies, resp)
}
func (ctx *DryRunContext) ReplyAsync(resp map[string]any) {
	ctx.replies = append(ctx.replies, fmt.Sprintf("%v", resp))
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// NormalizeDryRunHook 规范化并校验试跑的 Hook 名称（目前支持 OnMessage, OnConnect, OnInvoke；为空时默认 OnMessage）
func NormalizeDryRunHook(hook string) string {
	switch strings.TrimSpace(hook) {
	case "", OnMessage:
		return OnMessage
	case OnConnect, OnInvoke:
		return hook
	default:
		return ""
	}
}

// DryRun 是编解码脚本演练试跑的核心入口函数。
// 它在隔离的独立 Goja VM 中编译待测脚本（不会注册到全局 core.RegCodec，也不会替换线上正在运行的 VM 池），
// 执行指定的钩子函数（默认 OnMessage），并通过 EventBus 捕获执行期间的 console.log 调试日志。
//
// 返回的 DryRunResult 包含了日志、解析出的物模型属性、事件、设备会话下发动作等全方位诊断数据。
func DryRun(opts DryRunOpts) *DryRunResult {
	hook := NormalizeDryRunHook(opts.Hook)
	res := &DryRunResult{Hook: hook, Logs: []map[string]any{}, SavedProperties: []map[string]any{}, SavedEvents: []DryRunEvent{}, DeviceOnline: []string{}, SessionWrites: []DryRunWrite{}, Replies: []string{}}
	if hook == "" {
		res.Error = "hook must be OnMessage, OnConnect, or OnInvoke"
		return res
	}
	script := strings.TrimSpace(opts.Script)
	if script == "" {
		res.Error = "script is required"
		return res
	}
	productId := strings.TrimSpace(opts.ProductId)
	if productId == "" {
		productId = "dry-run"
	}
	deviceId := strings.TrimSpace(opts.DeviceId)
	if deviceId == "" {
		deviceId = "dry-run"
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = dryRunTimeout
	}

	pool, err := NewVmPool1(script, 1, productId)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer pool.Close()

	writes := []DryRunWrite{}
	sess := &dryRunSession{deviceId: deviceId, writes: &writes}
	dctx := &DryRunContext{
		BaseContext: core.BaseContext{ProductId: productId, DeviceId: deviceId, Session: sess},
		payload:     opts.Payload,
		topic:       opts.Topic,
		clientId:    opts.ClientId,
		username:    opts.Username,
		password:    opts.Password,
		sess:        sess,
	}
	if hook == OnInvoke {
		dctx.invoke = &core.FuncInvoke{
			FunctionId: opts.FunctionId,
			DeviceId:   deviceId,
			Data:       opts.Data,
		}
	}

	lis := eventbus.Listen(eventbus.GetDebugTopic(productId, "*"), 200)
	defer lis.Stop()
	sc := &ScriptCodec{script: script, productId: productId, pool: pool}
	_, err = sc.funcInvokeTimeout(hook, dctx, timeout)
	rawLogs := lis.Stop()
	res.Logs = debugMsgsToMaps(rawLogs)
	res.SavedProperties = dctx.savedProps
	res.SavedEvents = dctx.savedEvents
	res.DeviceOnline = dctx.online
	res.SessionWrites = writes
	res.Replies = dctx.replies
	if err == core.ErrFunctionNotImpl {
		res.FunctionMissing = true
		res.Error = err.Error()
		return res
	}
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Ok = true
	return res
}

// funcInvokeTimeout 在单次试跑中调用指定的 JS 函数，具备超时中断与 panic recover 保护
func (c *ScriptCodec) funcInvokeTimeout(name string, param interface{}, d time.Duration) (resp goja.Value, err error) {
	vm := c.pool.Get()
	if vm == nil {
		return nil, fmt.Errorf("codec pool closed for product: %s", c.productId)
	}
	defer c.pool.Put(vm)
	g := globeOf(vm)
	unbind := func() {}
	if g != nil {
		g.mu.Lock()
		defer g.mu.Unlock()
		unbind = g.bindDevice(param)
	}
	defer unbind()
	if d > 0 {
		timer := time.AfterFunc(d, func() { vm.Interrupt("dry-run timeout") })
		defer func() {
			timer.Stop()
			vm.ClearInterrupt()
		}()
	}
	fn, success := goja.AssertFunction(vm.Get(name))
	if !success {
		return nil, core.ErrFunctionNotImpl
	}
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%v", rec)
			resp = goja.Undefined()
			core.DebugLog("error", extractDeviceId(param), c.productId, err.Error())
			logs.Errorf("productId: [%s] dry-run script error: %v", c.productId, rec)
		}
	}()
	resp, err = fn(goja.Undefined(), vm.ToValue(param))
	if err != nil {
		core.DebugLog("error", extractDeviceId(param), c.productId, err.Error())
	}
	return resp, err
}

// debugMsgsToMaps 将 EventBus 接收到的调试消息转换为可被 JSON 序列化的 Map 列表
func debugMsgsToMaps(msgs []eventbus.Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		d, ok := m.(*eventbus.DebugMessage)
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"time": d.CreateTime, "level": d.Level, "deviceId": d.DeviceId, "data": d.Data,
		})
	}
	return out
}
