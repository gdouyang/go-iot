package boot

// sys start linstener
var sysStartListener []func()

func AddStartLinstener(call func()) {
	sysStartListener = append(sysStartListener, call)
}

func CallStartLinstener() {
	for _, call := range sysStartListener {
		call()
	}
}
