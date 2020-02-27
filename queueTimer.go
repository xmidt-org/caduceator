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
	queueEmptiedTime time.Time
	queueDepth       int
}

type Content struct {
	Status string
	Data   Data
}

type Data struct {
	ResultType string
	Result     []Result
}

type Result struct {
	Metric Metric
	Value  []interface{}
}

type Metric struct {
	Url string
}

func (app *App) doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func (app *App) startTimer() queueTime {

	var times queueTime
	//need to utilize prometheus

	logging.Info(app.logger).Log(logging.MessageKey(), "TIMER STARTED!")

	res, err := http.Get("http://prometheus:9090/api/v1/query?query=sum(xmidt_caduceus_outgoing_queue_depths)%20by%20(url)")
	currentTime := time.Now()
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "failed to query prometheus", logging.ErrorKey(), err.Error())
	} else {
		defer res.Body.Close()

		contents, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logging.Error(app.logger).Log(logging.MessageKey(), "failed to read body", logging.ErrorKey(), err.Error())
		}
		logging.Info(app.logger).Log(logging.MessageKey(), string(contents))

		var content Content
		json.Unmarshal([]byte(contents), &content)

		if content.Data.ResultType == "vector" {
			for _, results := range content.Data.Result {
				if results.Metric.Url == "http://caduceator:5000/events" && results.Value[1] == "0" {
					times.queueEmptiedTime = currentTime
					times.queueDepth = 0
					logging.Info(app.logger).Log(logging.MessageKey(), "TIME QUEUE GOT EMPTY (QUEUE TIME): "+times.queueEmptiedTime.String())

				}
			}
		}

		// logging.Info(app.logger).Log(logging.MessageKey(), "PARSING JSON: "+content.Data.Result[0].Value[1].(string))

		// logging.Info(app.logger).Log(logging.MessageKey(), string(contents))
		// queueDepth = content.Data.Result[0].Value[1].(int)
		// queueEmptyTime := time.Now()
		// if queueDepth == 0 {
		// 	// app.queueTime.queueEmptiedTime = queueEmptyTime
		// 	times.queueEmptiedTime = queueEmptyTime
		// }
	}
	return times
}
