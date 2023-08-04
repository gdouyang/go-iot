package common

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

type JsonResp struct {
	Msg     string      `json:"message"`
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Code    int         `json:"-"` // 20x, 30x, 40x, 50x
}

func JsonRespOk() JsonResp {
	return JsonResp{Success: true, Code: 200}
}

func JsonRespOkData(data interface{}) JsonResp {
	return JsonResp{Success: true, Result: data, Code: 200}
}

func JsonRespError(err error) JsonResp {
	return JsonResp{Success: false, Msg: err.Error(), Code: 400}
}

func JsonRespError1(err error, code int) JsonResp {
	return JsonResp{Success: false, Msg: err.Error(), Code: code}
}

func JsonRespErr(err *Err) JsonResp {
	return JsonResp{Success: false, Msg: err.Message, Code: err.Code}
}

type Err struct {
	Code    int
	Message string
}

func NewErr(code int, message string) *Err {
	return &Err{Code: code, Message: message}
}

// 请求错误
func NewErr400(message string) *Err {
	return NewErr(400, message)
}

// 内部错误
func NewErr500(message string) *Err {
	return NewErr(500, message)
}

// 超时
func NewErr504(message string) *Err {
	return NewErr(504, message)
}
