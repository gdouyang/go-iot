package north

import (
	northserver "go-iot/provider/north/server"
)

func echoToBrower(msg string) {
	northserver.EchoToBrower(msg)
}
