package registry

import (
	_ "go-iot/pkg/network/servers/http"
	_ "go-iot/pkg/network/servers/mqtt"
	_ "go-iot/pkg/network/servers/tcp"
	_ "go-iot/pkg/network/servers/websocket"
)
