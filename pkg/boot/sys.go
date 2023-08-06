package boot

import "go-iot/pkg/logger"

// sys start linstener
var sysStartListener []func()

func AddStartLinstener(call func()) {
	sysStartListener = append(sysStartListener, call)
}

func CallStartLinstener() {
	logger.Infof("sys start listener begin")
	for _, call := range sysStartListener {
		call()
	}
	logger.Infof("sys start listener end")
}
