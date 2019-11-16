package north

import (
	northserver "go-iot/provider/north/server"
)

func echoToBrower(msg string) {
	northserver.EchoToBrower(northserver.EchoMsg{Msg: msg, Type: "restful"})
}
