package registry

import (
	_ "go-iot/network/servers/http"
	_ "go-iot/network/servers/mqtt"
	_ "go-iot/network/servers/tcp"
	_ "go-iot/network/servers/websocket"
)
