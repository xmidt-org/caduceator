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
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	vegeta "github.com/tsenart/vegeta/lib"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

type Measures struct {
	TimeInMemory metrics.Histogram
}

// App used for logging and saving durations
type App struct {
	logger            log.Logger
	durations         chan time.Duration
	measures          *Measures
	attacker          *vegeta.Attacker
	counter           int
	maxRoutines       int
	mutex             *sync.Mutex
	queryURL          string
	queryExpression   string
	metricsURL        string
	sleepTime         time.Duration
	sleepTimeAfter    time.Duration
	prometheusAuth    string
	timeoutPrometheus time.Duration
}

const (
	TimeInMemory = "queue_empty_duration"
)

func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name:      TimeInMemory,
			Help:      "The duration it takes to empty queue in Caduceus.",
			Type:      "histogram",
			Namespace: "xmidt",
			Subsystem: "caduceator",
			Buckets:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
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

	time.Sleep(app.sleepTime)
	writer.WriteHeader(http.StatusAccepted)
	time.Sleep(app.sleepTimeAfter)

	_, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "Could not read request body", logging.ErrorKey(), err.Error())
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (app *App) receiveCutoff(writer http.ResponseWriter, req *http.Request) {

	cutoffTime := time.Now()

	logging.Info(app.logger).Log(logging.MessageKey(), "time caduceus queue is full: "+cutoffTime.String())

	logging.Info(app.logger).Log(logging.MessageKey(), "counter: "+strconv.Itoa(int(app.counter)))
	logging.Info(app.logger).Log(logging.MessageKey(), "max routines: "+strconv.Itoa(int(app.maxRoutines)))

	app.mutex.Lock()

	if app.counter <= app.maxRoutines {
		if app.counter == app.maxRoutines {
			app.attacker.Stop()
		}
		app.counter++
		go app.calculateDuration(cutoffTime)
		app.mutex.Unlock()

	}

	return
}
