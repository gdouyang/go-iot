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

import {pointer, marshalString, unmarshalString, marshalAllHeader, unmarshalAllHeader, marishalData, unmarshalData, unmarshalStringArray} from './marshal'

@external("easegress", "host_req_get_client_id") declare function host_req_get_client_id(): pointer;
export function getClientId(): string {
	let ptr = host_req_get_client_id()
	return unmarshalString(ptr)
}


@external("easegress", "host_req_get_user_name") declare function host_req_get_user_name(): pointer;
export function getUserName(): string {
	let ptr = host_req_get_user_name()
	return unmarshalString(ptr)
}

@external("easegress", "host_req_get_payload_string") declare function host_req_get_payload_string(): pointer;
export function getPayloadString(): string {
	let ptr = host_req_get_payload_string()
	return unmarshalString(ptr)
}
