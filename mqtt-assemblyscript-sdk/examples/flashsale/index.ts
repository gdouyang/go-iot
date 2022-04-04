export * from '../../easegress/proxy'

import { Program, parseDate, rand, getUnixTimeInMs, registerProgramFactory, client, log, LogLevel } from '../../easegress'
import { JSON } from "assemblyscript-json";
class FlashSale extends Program {

	constructor(params: Map<string, string>) {
		super(params)
    log(LogLevel.Info, params.toString())
	}

	run(): i32 {
		log(LogLevel.Info, "wasm====" + getUnixTimeInMs().toString())
    const str = client.getPayloadString()
		log(LogLevel.Info, "payload ===" + str)
    let d = (<JSON.Obj>JSON.parse(str));
    log(LogLevel.Info, "deviceName ===" + d.getString('deviceName')!.stringify())
		return 2
	}
}

registerProgramFactory((params: Map<string, string>) => {
	return new FlashSale(params)
})
