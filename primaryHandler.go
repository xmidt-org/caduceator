// SPDX-FileCopyrightText: 2020 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0


package main

import (
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/go-kit/log"
	vegeta "github.com/tsenart/vegeta/lib"
	"github.com/xmidt-org/webpa-common/v2/logging"  // nolint: staticcheck
	"github.com/xmidt-org/webpa-common/v2/xmetrics" // nolint: staticcheck
)

type Measures struct {
	TimeInMemory metrics.Histogram
}

// App used for logging and saving durations
type App struct {
	logger            log.Logger
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
	webhookURLs       []string
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
			Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
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

	_, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		logging.Error(app.logger).Log(logging.MessageKey(), "Could not read request body", logging.ErrorKey(), err.Error())
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

func (app *App) receiveCutoff(writer http.ResponseWriter, req *http.Request) {
	cutoffTime := time.Now()

	logging.Info(app.logger).Log(logging.MessageKey(), "time caduceus queue is full: "+cutoffTime.String())
	logging.Info(app.logger).Log(logging.MessageKey(), "counter: "+strconv.Itoa(app.counter))
	logging.Info(app.logger).Log(logging.MessageKey(), "max routines: "+strconv.Itoa(app.maxRoutines))

	app.mutex.Lock()

	if app.maxRoutines == 0 {
		app.counter++
		go app.calculateDuration(cutoffTime)
		app.mutex.Unlock()
	} else if app.counter <= app.maxRoutines {
		if app.counter == app.maxRoutines {
			app.attacker.Stop()
		}
		app.counter++
		go app.calculateDuration(cutoffTime)
		app.mutex.Unlock()
	}
}
