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
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

// App used for logging and saving durations

type Measures struct {
	TimeInMemory metrics.Histogram
}

type timeTracker interface {
	TrackTime(time.Duration)
}

type App struct {
	logger      log.Logger
	durations   chan time.Duration
	timeTracker timeTracker
	measures    *Measures
}

const (
	TimeInMemory = "xmidt_caduceator_queue_empty_duration"
)

func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name:      TimeInMemory,
			Help:      "The duration it takes to empty queue in Caduceus.",
			Type:      "histogram",
			Namespace: "xmidt",
			Subsystem: "caduceator",
		},
	}
}

func NewMeasures(p provider.Provider) *Measures {
	return &Measures{
		TimeInMemory: p.NewHistogram(TimeInMemory, 10),
	}
}

func (m *Measures) TrackTime(length time.Duration) {
	m.TimeInMemory.Observe(length.Seconds())
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
		// app.timeTracker.TrackTime(duration)
		app.measures.TimeInMemory.Observe(duration.Seconds())
		logging.Info(app.logger).Log(logging.MessageKey(), "SENT METRIC TO PROMETHEUS")
	}

	return
}
