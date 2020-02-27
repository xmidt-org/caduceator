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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/xmidt-org/webpa-common/logging"
)

type queueTime struct {
	// queueTime        chan time.Time
	cutoffTime       time.Time
	queueEmptiedTime time.Time
}

type Content struct {
	status string
	data   Data
}

type Data struct {
	resultType string
	result     Result
}

type Result struct {
	metric Metric
	value  []interface{}
}

type Metric struct {
	url string
}

func (app *App) startTimer() queueTime {

	var timeChannel queueTime
	//need to utilize prometheus

	logging.Info(app.logger).Log(logging.MessageKey(), "TIMER STARTED!")

	res, err := http.Get("http://prometheus:9090/api/v1/query?query=sum(xmidt_caduceus_outgoing_queue_depths)%20by%20(url)")
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "failed to query prometheus", logging.ErrorKey(), err.Error())
	} else {
		defer res.Body.Close()
		contents, err := ioutil.ReadAll(res.Body)
		var content Content
		json.Unmarshal([]byte(contents), &content)
		logging.Info(app.logger).Log(logging.MessageKey(), "PARSING JSON: "+content.status)
		if err != nil {
			logging.Error(app.logger).Log(logging.MessageKey(), "failed to read body", logging.ErrorKey(), err.Error())
		}
		logging.Info(app.logger).Log(logging.MessageKey(), string(contents))
		queueDepth := 0
		queueEmptyTime := time.Now()
		if queueDepth == 0 {
			app.queueTime.queueEmptiedTime = queueEmptyTime
		}
	}
	return timeChannel
}
