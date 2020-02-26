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
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/webpa-common/logging"
)

//Measure used for metrics
// type Measure struct {
// 	metric metrics.Counter
// }

//App used for logging and metrics
type App struct {
	logger     log.Logger
	channel    timeChannel
	cutoffTime time.Time
}

func (app *App) receiveEvents(writer http.ResponseWriter, req *http.Request) {
	_, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "Could not read request body", logging.ErrorKey(), err.Error())
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	time.Sleep(5 * time.Second)
	writer.WriteHeader(http.StatusAccepted)
}

func (app *App) receiveCutoff(writer http.ResponseWriter, req *http.Request) {

	var (
		buffer bytes.Buffer
	)

	req, err := http.NewRequest("GET", "http://localhost:9090/api/v1/query?query=xmidt_caduceus_outgoing_queue_depths", &buffer)
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "failed to create new request", logging.ErrorKey(), err.Error())
		return
	}

	//unmarshal json and parse information to new variable that will be inserted to channel in App struct

	app.channel.queueTime <- time.Now()
	app.cutoffTime = time.Now()
	// app.channel.queueTime = make(chan time.Time)
	app.channel = startTimer()
	return
	//stop registering for events
}
