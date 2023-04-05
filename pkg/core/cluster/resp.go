package cluster

type JsonResp struct {
	Msg     string      `json:"message"`
	Success bool        `json:"success"`
	Data    interface{} `json:"result,omitempty"`
	Code    int         `json:"-"` // 20x, 30x, 40x, 50x
}

func JsonRespOk() JsonResp {
	return JsonResp{Success: true, Code: 200}
}

func JsonRespOkData(data interface{}) JsonResp {
	return JsonResp{Success: true, Data: data, Code: 200}
}

func JsonRespError(err error) JsonResp {
	return JsonResp{Success: false, Msg: err.Error(), Code: 400}
}

func JsonRespError1(err error, code int) JsonResp {
	return JsonResp{Success: false, Msg: err.Error(), Code: code}
}
