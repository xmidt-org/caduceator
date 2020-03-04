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
	"crypto/sha1"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	vegeta "github.com/tsenart/vegeta/lib"
	acquire "github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/webpa-common/basculechecks"
	"github.com/xmidt-org/webpa-common/basculemetrics"
	"github.com/xmidt-org/webpa-common/concurrent"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/server"
	"github.com/xmidt-org/wrp-go/wrp"
	webhook "github.com/xmidt-org/wrp-listener"
	"github.com/xmidt-org/wrp-listener/hashTokenFactory"
	secretGetter "github.com/xmidt-org/wrp-listener/secret"
	"github.com/xmidt-org/wrp-listener/webhookClient"
)

// Register used to start and stop registering webhooks

const (
	applicationName = "caduceator"
)

type Config struct {
	VegetaConfig     VegetaConfig
	Webhook          Webhook
	Secret           Secret
	PrometheusConfig PrometheusConfig
	MetricConfig     MetricConfig
}

type VegetaConfig struct {
	Frequency   int
	Connections int
	Duration    time.Duration
	MaxRoutines int
}

type Request struct {
	WebhookConfig WebhookConfig
	Events        string
}

type WebhookConfig struct {
	URL           string
	FailureURL    string
	Secret        string
	MaxRetryCount int
}

type Webhook struct {
	RegistrationInterval string
	Timeout              int
	RegistrationURL      string
	Request              Request
	Basic                string
	JWT                  JWT
}

type Secret struct {
	Header    string
	Delimiter string
}

type JWT struct {
	AuthURL string
	Timeout string
	Buffer  string
}

type PrometheusConfig struct {
	QueryURL        string
	QueryExpression string
}

type MetricConfig struct {
	Address    string
	MetricsURL string
}

// Start function is used to send events to Caduceus
func Start(id uint64, acquirer *acquire.FixedValueAcquirer, logger log.Logger) vegeta.Targeter {

	return func(target *vegeta.Target) (err error) {

		message := wrp.Message{
			Type:            4,
			Source:          "dns:talaria",
			Destination:     "event:device-status/mac:112233445566/offline",
			TransactionUUID: "abcd",
			ContentType:     "json",
			Metadata: map[string]string{
				"/trust":      "0",
				"/compliance": "full",
				"/boot-time":  "1582511208",
			},
			Payload: []byte("ewoJCSJpZCI6ICJtYWM6MTEyMjMzNDQ1NTY2IiwKCQkidHMiOiAiMjAyMC0wMi0yMFQwMToyMjoyNC4xNDc0NDkzMDJaIiwKCQkiYnl0ZXMtc2VudCI6IDIxMzQsCgkJIm1lc3NhZ2VzLXNlbnQiOiA4LAoJCSJieXRlcy1yZWNlaXZlZCI6IDU1NTcsCgkJIm1lc3NhZ2VzLXJlY2VpdmVkIjogMSwKCQkiY29ubmVjdGVkLWF0IjogIjIwMjAtMDItMTlUMTE6NTU6MjMuNzQ1MDU1NzFaIiwKCQkicmVhc29uLWZvci1jbG9zdXJlIjogInJlYWRlcnJvciIKCX0="),
		}

		// encoding wrp.Message
		var (
			buffer  bytes.Buffer
			encoder = wrp.NewEncoder(&buffer, wrp.Msgpack)
		)

		if err := encoder.Encode(&message); err != nil {
			logging.Error(logger).Log(logging.MessageKey(), "failed to encode payload", logging.ErrorKey(), err.Error())
			return err
		}

		req, err := http.NewRequest("POST", "http://caduceus:6000/api/v3/notify", &buffer)
		if err != nil {
			logging.Error(logger).Log(logging.MessageKey(), "failed to create new request", logging.ErrorKey(), err.Error())
			return err
		}
		req.Header.Add("Content-type", "application/msgpack")

		authValue, err := acquirer.Acquire()
		if err != nil {
			logging.Error(logger).Log(logging.MessageKey(), "failed to acquire", logging.ErrorKey(), err.Error())
			return err
		}

		req.Header.Add("Authorization", authValue)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logging.Error(logger).Log(logging.MessageKey(), "failed while making HTTP request", logging.ErrorKey(), err.Error())
			return err
		}
		defer resp.Body.Close()

		return err
	}

}

func main() {

	var (
		f, v                                     = pflag.NewFlagSet(applicationName, pflag.ContinueOnError), viper.New()
		logger, metricsRegistry, caduceator, err = server.Initialize(applicationName, os.Args, f, v, basculechecks.Metrics, basculemetrics.Metrics, Metrics)
	)

	config := new(Config)
	v.Unmarshal(config)

	// use constant secret for hash
	secretGetter := secretGetter.NewConstantSecret(config.Webhook.Request.WebhookConfig.Secret)

	// set up the middleware
	htf, err := hashTokenFactory.New("Sha1", sha1.New, secretGetter)
	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "failed to setup hash token factory", logging.ErrorKey(), err.Error())
		os.Exit(1)
	}
	authConstructor := basculehttp.NewConstructor(
		basculehttp.WithTokenFactory("Sha1", htf),
		basculehttp.WithHeaderName(config.Secret.Header),
		basculehttp.WithHeaderDelimiter(config.Secret.Delimiter),
	)
	eventHandler := alice.New(authConstructor)
	cutoffHandler := alice.New()

	// set up the registerer

	basicConfig := webhookClient.BasicConfig{
		Timeout:         5 * time.Second,
		RegistrationURL: config.Webhook.RegistrationURL,
		Request: webhook.W{
			Config: webhook.Config{
				URL: config.Webhook.Request.WebhookConfig.URL,
			},
			Events:     []string{config.Webhook.Request.Events},
			FailureURL: config.Webhook.Request.WebhookConfig.FailureURL,
		},
	}

	acquirer, err := acquire.NewFixedAuthAcquirer(config.Webhook.Basic)
	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "failed to create basic auth plain text acquirer:", logging.ErrorKey(), err.Error())
		os.Exit(1)
	}

	registerer, err := webhookClient.NewBasicRegisterer(acquirer, secretGetter, basicConfig)
	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "failed to setup registerer", logging.ErrorKey(), err.Error())
		os.Exit(1)
	}
	periodicRegisterer := webhookClient.NewPeriodicRegisterer(registerer, 4*time.Minute, logger)

	// start the registerer
	periodicRegisterer.Start()

	router := mux.NewRouter()

	measures := NewMeasures(metricsRegistry)

	attacker := vegeta.NewAttacker(vegeta.Connections(config.VegetaConfig.Connections))

	durations := make(chan time.Duration, config.VegetaConfig.MaxRoutines)

	app := &App{logger: logger,
		measures:        measures,
		attacker:        attacker,
		maxRoutines:     config.VegetaConfig.MaxRoutines,
		counter:         0,
		durations:       durations,
		mutex:           &sync.Mutex{},
		queryURL:        config.PrometheusConfig.QueryURL,
		queryExpression: config.PrometheusConfig.QueryExpression,
		metricsURL:      config.MetricConfig.MetricsURL,
	}

	// start listening
	logging.Info(logger).Log(logging.MessageKey(), "before handler")

	router.Handle("/events", eventHandler.ThenFunc(app.receiveEvents)).Methods("POST")
	router.Handle("/cutoff", cutoffHandler.ThenFunc(app.receiveCutoff)).Methods("POST")

	logging.Info(logger).Log(logging.MessageKey(), "after handler")

	_, runnable, done := caduceator.Prepare(logger, nil, metricsRegistry, router)
	waitGroup, shutdown, err := concurrent.Execute(runnable)
	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "failed to execute additional process", logging.ErrorKey(), err.Error())
	}

	// send events to Caduceus using vegeta
	var metrics vegeta.Metrics
	rate := vegeta.Rate{Freq: config.VegetaConfig.Frequency, Per: time.Second}
	duration := config.VegetaConfig.Duration * time.Minute

	for res := range attacker.Attack(Start(0, acquirer, logger), rate, duration, "Big Bang!") {
		metrics.Add(res)
	}

	metricsReporter := vegeta.NewTextReporter(&metrics)

	err = metricsReporter.Report(os.Stdout)

	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "vegeta failed", logging.ErrorKey(), err.Error())
	}

	signals := make(chan os.Signal, 10)
	signal.Notify(signals)
	for exit := false; !exit; {
		select {
		case s := <-signals:
			if s != os.Kill && s != os.Interrupt {
				logging.Info(logger).Log(logging.MessageKey(), "ignoring signal", "signal", s)
			} else {
				logging.Error(logger).Log(logging.MessageKey(), "exiting due to signal", "signal", s)
				exit = true
			}
		case <-done:
			logging.Error(logger).Log(logging.MessageKey(), "one or more servers exited")
			exit = true
		}
	}

	metrics.Close()
	close(shutdown)
	waitGroup.Wait()
	logging.Info(logger).Log(logging.MessageKey(), "Caduceator has shut down")

}
