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
	"net/url"
	"time"

	"github.com/xmidt-org/webpa-common/logging"
)

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

	logging.Info(app.logger).Log(logging.MessageKey(), "entered duration function")

	encodedQuery := &url.URL{Path: app.queryExpression}

	req, err := http.NewRequest("GET", app.queryURL+"?query="+encodedQuery.String(), nil)
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "failed to get prometheus url", logging.ErrorKey(), err.Error())
	}

	req.Header.Add("Authorization", app.prometheusAuth)

Loop:
	for {

		currentTime := time.Now()

		res, err := http.DefaultClient.Do(req)

		if err != nil {
			logging.Error(app.logger).Log(logging.MessageKey(), "failed to query prometheus", logging.ErrorKey(), err.Error())
			return
		} else {
			defer res.Body.Close()

			contents, err := ioutil.ReadAll(res.Body)
			if err != nil {
				logging.Error(app.logger).Log(logging.MessageKey(), "failed to read body", logging.ErrorKey(), err.Error())
			}

			var content Content
			json.Unmarshal([]byte(contents), &content)

			if content.Data.ResultType == "vector" {
				for _, results := range content.Data.Result {

					// only calculating duration once queue size reaches 0
					if results.Metric.Url == app.metricsURL && results.Value[1] == "0" {

						// putting calculated duration into histogram metric
						app.measures.TimeInMemory.Observe(currentTime.Sub(cutoffTime).Seconds())

						logging.Info(app.logger).Log(logging.MessageKey(), "time queue is 0: "+currentTime.String())

						logging.Info(app.logger).Log(logging.MessageKey(), "sent histogram metric to prometheus: "+currentTime.Sub(cutoffTime).String())
						break Loop
					}
				}
			}
		}
	}
	return
}
