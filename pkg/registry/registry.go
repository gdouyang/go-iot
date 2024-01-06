// 注册所有init方法
package registry

import (
	// web api
	_ "go-iot/pkg/api"
	// 编解码
	_ "go-iot/pkg/codec"

	// timeseries
	_ "go-iot/pkg/timeseries"

	// servers
	_ "go-iot/pkg/network/servers/http"
	_ "go-iot/pkg/network/servers/mqtt"
	_ "go-iot/pkg/network/servers/tcp"
	_ "go-iot/pkg/network/servers/websocket"

	// clients
	_ "go-iot/pkg/network/clients/modbus"
	_ "go-iot/pkg/network/clients/mqtt"
	_ "go-iot/pkg/network/clients/tcp"

	// notifys
	_ "go-iot/pkg/notify/dingtalk"
	_ "go-iot/pkg/notify/email"
	_ "go-iot/pkg/notify/webhook"
)
