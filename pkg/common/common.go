// 公共方法、类型
package common

import "net/http"

// json格式的响应
type JsonResp struct {
	Msg     string      `json:"message"`
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Code    int         `json:"-"` // 20x, 30x, 40x, 50x
}

func JsonRespOk() JsonResp {
	return JsonResp{Success: true, Code: http.StatusOK}
}

func JsonRespOkData(data interface{}) JsonResp {
	return JsonResp{Success: true, Result: data, Code: http.StatusOK}
}

func JsonRespError(err error) JsonResp {
	return JsonResp{Success: false, Msg: err.Error(), Code: http.StatusBadRequest}
}

func JsonRespError1(err error, code int) JsonResp {
	return JsonResp{Success: false, Msg: err.Error(), Code: code}
}

func JsonRespErr(err *Err) JsonResp {
	return JsonResp{Success: false, Msg: err.Message, Code: err.Code}
}

type Err struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewErr(code int, message string) *Err {
	return &Err{Code: code, Message: message}
}

func NewErrCode(code int) *Err {
	return &Err{Code: code, Message: http.StatusText(code)}
}

// 请求错误
func NewErr400(message string) *Err {
	return NewErr(http.StatusBadRequest, message)
}

// 内部错误
func NewErr500(message string) *Err {
	return NewErr(http.StatusInternalServerError, message)
}

// 超时
func NewErr504(message string) *Err {
	return NewErr(http.StatusGatewayTimeout, message)
}
