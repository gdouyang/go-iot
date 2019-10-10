package agent

type Pas struct {
	Result
	Request
}

type Result struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Data    string `json:"data"`
	Requuid string `json:"uuid"`
}

type Request struct {
	Action     string  `json:"action"`
	Rspuuid    string  `json:"uuid"`
	Desc       string  `json:"desc"`
	CameraCode string  `json:"cameraCode"`
	ImosCode   string  `json:"imosCode"`
	MediaCode  string  `json:"mediaCode"`
	RtspUrl    string  `json:"rtspUrl"`
	Resolution string  `json:"resolution"`
	StartTime  string  `json:"startTime"`
	EndTime    string  `json:"endTime"`
	FlowName   string  `json:"flowName"`
	SeekTime   string  `json:"seekTime"`
	SendTime   int64   `json:"sendTime"`
	SIP        string  `json:"serverIp"`    //当作为透传服务的时候的ip
	SPort      string  `json:"serverPort"`  //透传的服务端口
	SRequest   string  `json:"serverPort"`  //透传的消息
	SResponse  string  `json:"serverPort"`  //响应
	SOnvif     string  `json:"serverOnvif"` //onvif的服务ip端口
	UserOnvif  string  `json:"user"`        //onvif用户
	PassOnvif  string  `json:"pass"`        //onvif密码
	X          float32 `json:"x"`
	Y          float32 `json:"y"`
	Z          float32 `json:"z"`
	Preset     int     `json:"preset"` //预置位编号
}
