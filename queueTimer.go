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

type QueueTime struct {
	queueEmptiedTime time.Time
	cutoffTime       time.Time
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

func (app *App) calculateDuration(cutoffTime time.Time) {

	// make requests to get caduceus queue depth metrics
Loop:
	for {
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
			// logging.Info(app.logger).Log(logging.MessageKey(), string(contents))

			var content Content
			json.Unmarshal([]byte(contents), &content)

			if content.Data.ResultType == "vector" {
				for _, results := range content.Data.Result {

					//only calculating duration once queue size reaches 0
					if results.Metric.Url == "http://caduceator:5000/events" && results.Value[1] == "0" {

						//putting calculated duration into channel
						app.durations <- currentTime.Sub(cutoffTime)
						logging.Info(app.logger).Log(logging.MessageKey(), "PLACED DURATION IN CHANNEL! "+currentTime.Sub(cutoffTime).String())
						break Loop
					}
				}
			}
		}
	}
	return
}
