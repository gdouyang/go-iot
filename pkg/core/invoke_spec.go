package core

type MessageType string

const (
	FUNC_INVOKE = "FuncInvoke" // 功能调用
)

// 功能调用
type FuncInvoke struct {
	TraceId    string                 `json:"traceId"` // 跟踪ID
	FunctionId string                 `json:"functionId"`
	DeviceId   string                 `json:"deviceId"`
	ClusterId  string                 `json:"clusterId,omitempty"`
	Data       map[string]interface{} `json:"data"`
	Async      string                 `json:"async,omitempty"` // 是否异步执行，为"true"时将覆盖物模型的配置
	Timeout    int                    `json:"timeout"`         // 同步调用时指定timeout可以覆盖默认超时时间
	Replay     chan *FuncInvokeReply  `json:"-"`
}

func (p *FuncInvoke) Type() MessageType {
	return FUNC_INVOKE
}

type FuncInvokeReply struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg,omitempty"`
	TraceId string `json:"-"`
}
