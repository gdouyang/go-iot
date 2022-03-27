/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// This exports all functions required by Easegress
export * from '../../easegress/proxy'

import { Program, parseDate, rand, getUnixTimeInMs, registerProgramFactory, request, log, LogLevel } from '../../easegress'

class FlashSale extends Program {
	// startTime is the start time of the flash sale, unix timestamp in millisecond
	startTime: i64

	// blockRatio is the ratio of requests being blocked to protect backend service
	// for example: 0.4 means we blocks 40% of the requests randomly.
	blockRatio: f64

	// maxPermission is the upper limits of permitted users 
	maxPermission: i32

	constructor(params: Map<string, string>) {
		super(params)

		let key = "startTime"
		if (params.has(key)) {
			let val = params.get(key)
			this.startTime = parseDate(val).getTime()
		}

		key = "blockRatio"
		if (params.has(key)) {
			let val = params.get(key)
			this.blockRatio = parseFloat(val)
		}

		key = "maxPermission"
		if (params.has(key)) {
			let val = params.get(key)
			this.maxPermission = i32(parseInt(val))
		}
	}

	run(): i32 {
		log(LogLevel.Info, "wasm====" + getUnixTimeInMs().toString())
		return 2
	}
}

registerProgramFactory((params: Map<string, string>) => {
	return new FlashSale(params)
})
