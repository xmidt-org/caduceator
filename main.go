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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
}

type VegetaConfig struct {
	Frequency      int
	Period         time.Duration
	Connections    int
	Duration       time.Duration
	MaxRoutines    int
	PostURL        string
	SleepTime      time.Duration
	ClientTimeout  time.Duration
	SleepTimeAfter time.Duration
	WrpMessageDest string
	VegetaRehash   VegetaRehash
}

type VegetaRehash struct {
	Routines       int
	Frequency      int
	Period         time.Duration
	Connections    int
	Duration       time.Duration
	Sleep          time.Duration
	WrpMessageDest string
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
	RegistrationInterval time.Duration
	Timeout              time.Duration
	RegistrationURL      string
	WebhookCount         int
	Request              Request
	Basic                string
	JWT                  JWT
}

type Secret struct {
	Header    string
	Delimiter string
}

type JWT struct {
	RequestHeaders map[string]string
	AuthURL        string
	Timeout        time.Duration
	Buffer         time.Duration
}

type PrometheusConfig struct {
	QueryURL        string
	QueryExpression string
	MetricsURL      string
	Auth            string
	Timeout         time.Duration
}

func vegetaStarter(metrics vegeta.Metrics, config *Config, attacker *vegeta.Attacker, acquirer acquire.Acquirer, logger log.Logger) {
	rate := vegeta.Rate{Freq: config.VegetaConfig.Frequency, Per: config.VegetaConfig.Period}
	duration := config.VegetaConfig.Duration * time.Minute

	for res := range attacker.Attack(Start(0, acquirer, logger, config.VegetaConfig.PostURL, config.VegetaConfig.ClientTimeout, config.VegetaConfig.WrpMessageDest), rate, duration, "Big Bang!") {
		metrics.Add(res)
	}

	metricsReporter := vegeta.NewTextReporter(&metrics)

	err := metricsReporter.Report(os.Stdout)

	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "vegeta failed", logging.ErrorKey(), err.Error())
		os.Exit(1)
	}
}

func rehashStarter(metrics vegeta.Metrics, config *Config, attacker *vegeta.Attacker, acquirer acquire.Acquirer, logger log.Logger) {
	rate := vegeta.Rate{Freq: config.VegetaConfig.VegetaRehash.Frequency, Per: config.VegetaConfig.Period}
	duration := config.VegetaConfig.VegetaRehash.Duration * time.Minute

	for res := range attacker.Attack(Start(0, acquirer, logger, config.VegetaConfig.PostURL, config.VegetaConfig.ClientTimeout, config.VegetaConfig.VegetaRehash.WrpMessageDest), rate, duration, "Big Bang!") {
		metrics.Add(res)
	}

	metricsReporter := vegeta.NewTextReporter(&metrics)

	err := metricsReporter.Report(os.Stdout)

	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "vegeta failed", logging.ErrorKey(), err.Error())
		os.Exit(1)
	}
}

// Start function is used to send events to Caduceus
func Start(id uint64, acquirer acquire.Acquirer, logger log.Logger, requestURL string, timeout time.Duration, destination string) vegeta.Targeter {
	var client = &http.Client{
		Timeout: timeout,
	}
	return func(target *vegeta.Target) (err error) {

		message := wrp.Message{
			Type:            4,
			Source:          "dns:talaria",
			Destination:     destination,
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
		}

		req, err := http.NewRequest("POST", requestURL, &buffer)
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

		fmt.Println("AUTH VALUE: " + authValue)
		req.Header.Add("Authorization", authValue)
		resp, err := client.Do(req)

		if err != nil {
			logging.Error(logger).Log(logging.MessageKey(), "failed while making HTTP request: ", logging.ErrorKey(), err.Error())
			return err
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()

		return err
	}

}

func determineTokenAcquirer(wh Webhook) (acquire.Acquirer, error) {
	defaultAcquirer := &acquire.DefaultAcquirer{}
	if wh.JWT.AuthURL != "" && wh.JWT.Buffer != 0 && wh.JWT.Timeout != 0 {
		acquireConfig := acquire.RemoteBearerTokenAcquirerOptions{
			AuthURL:        wh.JWT.AuthURL,
			Timeout:        wh.JWT.Timeout,
			Buffer:         wh.JWT.Buffer,
			RequestHeaders: wh.JWT.RequestHeaders,
		}
		return acquire.NewRemoteBearerTokenAcquirer(acquireConfig)
	}

	if wh.Basic != "" {
		fmt.Println("basic!")
		fmt.Println(wh.Basic)
		return acquire.NewFixedAuthAcquirer(wh.Basic)
	}

	return defaultAcquirer, nil
}

func main() {

	var (
		f, v                                     = pflag.NewFlagSet(applicationName, pflag.ContinueOnError), viper.New()
		logger, metricsRegistry, caduceator, err = server.Initialize(applicationName, os.Args, f, v, basculechecks.Metrics, basculemetrics.Metrics, Metrics)
	)

	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "failed to initialize", logging.ErrorKey(), err.Error())
	}

	config := new(Config)
	err = v.Unmarshal(config)
	if err != nil {
		logging.Error(logger).Log(logging.MessageKey(), "failed to unmarshal config", logging.ErrorKey(), err.Error())
	}

	logging.Info(logger).Log(logging.MessageKey(), "vegeta frequency")
	logging.Info(logger).Log(logging.MessageKey(), config.VegetaConfig.Frequency)

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

	var acquirer acquire.Acquirer

	var webhookURLs []string

	periodicRegisterersList := make([]*webhookClient.PeriodicRegisterer, config.Webhook.WebhookCount)

	for i := 1; i <= config.Webhook.WebhookCount; i++ {
		// set up the registerer
		basicConfig := webhookClient.BasicConfig{
			Timeout: config.Webhook.Timeout,
			// RegistrationURL: config.Webhook.RegistrationURL + "?webhook=" + strconv.Itoa(i),
			RegistrationURL: config.Webhook.RegistrationURL,
			Request: webhook.W{
				Config: webhook.Config{
					URL: config.Webhook.Request.WebhookConfig.URL + "?webhook=" + strconv.Itoa(i),
				},
				Events:     []string{config.Webhook.Request.Events},
				FailureURL: config.Webhook.Request.WebhookConfig.FailureURL + "?webhook=" + strconv.Itoa(i),
			},
		}

		webhookURLs = append(webhookURLs, basicConfig.Request.Config.URL)

		acquirer, err = determineTokenAcquirer(config.Webhook)
		if err != nil {
			logging.Error(logger).Log(logging.MessageKey(), "failed to create auth plain text acquirer:", logging.ErrorKey(), err.Error())
			os.Exit(1)
		}

		registerer, err := webhookClient.NewBasicRegisterer(acquirer, secretGetter, basicConfig)
		if err != nil {
			logging.Error(logger).Log(logging.MessageKey(), "failed to setup registerer", logging.ErrorKey(), err.Error())
			os.Exit(1)
		}

		periodicRegisterer := webhookClient.NewPeriodicRegisterer(registerer, config.Webhook.RegistrationInterval, logger)
		periodicRegisterersList = append(periodicRegisterersList, periodicRegisterer)

		periodicRegisterer.Start()
	}

	router := mux.NewRouter()

	measures := NewMeasures(metricsRegistry)

	attacker := vegeta.NewAttacker(vegeta.Connections(config.VegetaConfig.Connections))

	app := &App{logger: logger,
		measures:          measures,
		attacker:          attacker,
		maxRoutines:       config.VegetaConfig.MaxRoutines,
		counter:           1,
		mutex:             &sync.Mutex{},
		queryURL:          config.PrometheusConfig.QueryURL,
		queryExpression:   config.PrometheusConfig.QueryExpression,
		metricsURL:        config.PrometheusConfig.MetricsURL,
		sleepTime:         config.VegetaConfig.SleepTime,
		sleepTimeAfter:    config.VegetaConfig.SleepTimeAfter,
		prometheusAuth:    config.PrometheusConfig.Auth,
		timeoutPrometheus: config.PrometheusConfig.Timeout,
		webhookURLs:       webhookURLs,
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
		os.Exit(1)
	}

	// send events to Caduceus using vegeta
	var metrics vegeta.Metrics

	go vegetaStarter(metrics, config, attacker, acquirer, logger)

	rehashTicker := time.NewTicker(config.VegetaConfig.VegetaRehash.Period * time.Minute)

	if config.VegetaConfig.VegetaRehash.Routines > 0 {
		for {
			select {
			case <-rehashTicker.C:
				for i := 0; i < config.VegetaConfig.VegetaRehash.Routines; i++ {
					go rehashStarter(metrics, config, attacker, acquirer, logger)
				}
			}
		}
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
	for i := 0; i < len(periodicRegisterersList); i++ {
		periodicRegisterersList[i].Stop()
	}
	close(shutdown)
	waitGroup.Wait()
	logging.Info(logger).Log(logging.MessageKey(), "Caduceator has shut down")

}
