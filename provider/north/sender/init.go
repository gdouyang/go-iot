package sender

import (
	"go-iot/models"
	northserver "go-iot/provider/north/server"
	"go-iot/provider/util"
)

func echoToBrower(req models.IotRequest) {
	data, _ := util.JsonEncoderHTML(req)
	northserver.EchoToBrower(northserver.EchoMsg{Msg: string(data), Type: "restful"})
}
