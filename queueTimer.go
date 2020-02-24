/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
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
 *
 */

package main

import (
	"time"
)

type timeChannel struct {
	queueTime chan time.Time
}

func startTimer() chan time.Time {
	var timeChannel timeChannel
	//need to utilize prometheus
	var time time.Time
	//add prometheus code here to check time and put into channel
	timeChannel.queueTime <- time
	return timeChannel.queueTime
}
