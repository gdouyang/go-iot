package registry

import (
	_ "go-iot/pkg/network/servers/http"
	_ "go-iot/pkg/network/servers/mqtt"
	_ "go-iot/pkg/network/servers/tcp"
	_ "go-iot/pkg/network/servers/websocket"

	_ "go-iot/pkg/network/clients/modbus"
	_ "go-iot/pkg/network/clients/mqtt"
	_ "go-iot/pkg/network/clients/tcp"

	_ "go-iot/pkg/notify/dingtalk"
	_ "go-iot/pkg/notify/email"
	_ "go-iot/pkg/notify/webhook"
)
