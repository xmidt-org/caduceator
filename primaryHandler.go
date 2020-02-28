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
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/webpa-common/logging"
)

// App used for logging and saving durations
type App struct {
	logger    log.Logger
	durations chan time.Duration
}

func (app *App) receiveEvents(writer http.ResponseWriter, req *http.Request) {
	logging.Info(app.logger).Log(logging.MessageKey(), "CADUCEUS STARTED RECEIVING EVENTS!")

	_, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "Could not read request body", logging.ErrorKey(), err.Error())
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	writer.WriteHeader(http.StatusAccepted)
}

func (app *App) receiveCutoff(writer http.ResponseWriter, req *http.Request) {

	logging.Info(app.logger).Log(logging.MessageKey(), "CADUCEUS QUEUE IS FULL!")

	cutoffTime := time.Now()

	app.durations = make(chan time.Duration)
	go app.calculateDuration(cutoffTime)

	// add here for putting duration into metrics histogram
	for duration := range app.durations {
		logging.Info(app.logger).Log(logging.MessageKey(), "DURATION FROM CHANNEL: "+duration.String())
	}

	return
}
