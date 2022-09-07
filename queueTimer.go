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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/xmidt-org/webpa-common/v2/logging"
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (app *App) calculateDuration(cutoffTime time.Time) {

	logging.Info(app.logger).Log(logging.MessageKey(), "entered duration function")

	var client = &http.Client{
		Timeout: app.timeoutPrometheus,
	}

	encodedQuery := url.QueryEscape(app.queryExpression)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?query=%s", app.queryURL, encodedQuery), nil)
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "failed to get prometheus url", logging.ErrorKey(), err.Error())
	}

	req.Header.Add("Authorization", app.prometheusAuth)
	logging.Info(app.logger).Log(logging.MessageKey(), "added authorization")

	for {

		currentTime := time.Now()

		res, err := client.Do(req)

		if err != nil {
			logging.Error(app.logger).Log(logging.MessageKey(), "failed to query prometheus", logging.ErrorKey(), err.Error())
			return
		} else {
			defer res.Body.Close()

			contents, err := io.ReadAll(res.Body)
			if err != nil {
				logging.Error(app.logger).Log(logging.MessageKey(), "failed to read body", logging.ErrorKey(), err.Error())
			}

			var content Content
			if err := json.Unmarshal(contents, &content); err != nil {
				logging.Error(app.logger).Log(logging.MessageKey(), "unable to unmarshal prometheus query body", logging.ErrorKey(), err, "contents", string(contents))
				return
			}

			if content.Data.ResultType == "vector" {
				for _, results := range content.Data.Result {
					// only calculating duration once queue size reaches 0
					val, _ := strconv.Atoi(results.Value[1].(string))
					if contains(app.webhookURLs, results.Metric.Url) && val <= 500 {
						// putting calculated duration into histogram metric
						app.measures.TrackTime(currentTime.Sub(cutoffTime))

						logging.Info(app.logger).Log(logging.MessageKey(), "time queue is 0: "+currentTime.String())

						logging.Info(app.logger).Log(logging.MessageKey(), "sent histogram metric to prometheus: "+currentTime.Sub(cutoffTime).String())
						return
					}
				}
			}
		}
	}
}
