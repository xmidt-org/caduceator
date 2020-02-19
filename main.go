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
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/justinas/alice"
	vegeta "github.com/tsenart/vegeta/lib"
	acquire "github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/bascule/basculehttp"
	webhook "github.com/xmidt-org/wrp-listener"
	"github.com/xmidt-org/wrp-listener/hashTokenFactory"
	secretGetter "github.com/xmidt-org/wrp-listener/secret"
	"github.com/xmidt-org/wrp-listener/webhookClient"
)

type Register struct {
	periodicRegisterer *webhookClient.PeriodicRegisterer
}

//Start function is used to send events to Caduceus
func Start(id uint64) vegeta.Targeter {

	return func(target *vegeta.Target) (err error) {

		//add code in here to send events to Caduceus (use http lib to make request; refer to example curl command)
		file, err := os.Open("./payload")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open payload: %v\n", err.Error())
		}
		defer file.Close()

		req, err := http.NewRequest("POST", "https://caduceus-cd.xmidt.comcast.net:443/api/v3/notify", file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create new request: %v\n", err.Error())
		}
		req.Header.Add("Content-type", "application/msgpack")

		acquirer, err := acquire.NewFixedAuthAcquirer("Basic " + base64.StdEncoding.EncodeToString([]byte("dXNlcjpwYXNz")))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create basic auth plain text acquirer: %v\n", err.Error())
			os.Exit(1)
		}

		authValue, err := acquirer.Acquire()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to acquire: %v\n", err.Error())
		}

		req.Header.Add("Authorization", authValue)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed while making HTTP request: %v\n", err.Error())
		}
		defer resp.Body.Close()

		return err
	}

}

func main() {

	// use constant secret for hash
	secretGetter := secretGetter.NewConstantSecret("secret1234")

	// set up the middleware
	htf, err := hashTokenFactory.New("Sha1", sha1.New, secretGetter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup hash token factory: %v\n", err.Error())
		os.Exit(1)
	}
	authConstructor := basculehttp.NewConstructor(
		basculehttp.WithTokenFactory("Sha1", htf),
		basculehttp.WithHeaderName("X-Webpa-Signature"),
		basculehttp.WithHeaderDelimiter("="),
	)
	handler := alice.New(authConstructor)

	// set up the registerer
	basicConfig := webhookClient.BasicConfig{
		Timeout:         5 * time.Second,
		RegistrationURL: "http://127.0.0.1:6000/hook",
		Request: webhook.W{
			Config: webhook.Config{
				URL: "http://127.0.0.1:5000/events",
			},
			Events:     []string{"device-status.*"},
			FailureURL: "http://127.0.0.1:5000/events",
		},
	}

	acquirer, err := acquire.NewFixedAuthAcquirer("Basic " + base64.StdEncoding.EncodeToString([]byte("dXNlcjpwYXNz")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create basic auth plain text acquirer: %v\n", err.Error())
		os.Exit(1)
	}

	registerer, err := webhookClient.NewBasicRegisterer(acquirer, secretGetter, basicConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup registerer: %v\n", err.Error())
		os.Exit(1)
	}
	periodicRegisterer := webhookClient.NewPeriodicRegisterer(registerer, 4*time.Minute, nil)

	// start the registerer
	periodicRegisterer.Start()

	// start listening
	http.Handle("/events", handler.ThenFunc(return200)) //need to change func
	err = http.ListenAndServe(":5000", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serving http requests: %v\n", err.Error())
		os.Exit(1)
	}

	//still need to call both function in primaryHandler using router.Handle
	//will get time the queue was empty from channel
	emptyQueueTime := <-channel
	elapsedTime := emptyQueueTime.Sub(cutoffTime)
	println(elapsedTime)

	//send events to Caduseus using vegeta
	var metrics vegeta.Metrics
	rate := vegeta.Rate{Freq: 100, Per: time.Second}
	duration := 1 * time.Second

	// targeter := vegeta.NewStaticTargeter(vegeta.Target{
	// 	Method: "GET",
	// 	URL:    "http://localhost:9100/", //need to change URL
	// })

	attacker := vegeta.NewAttacker(vegeta.Connections(500))

	for res := range attacker.Attack(Start(0), rate, duration, "Big Bang!") {
		metrics.Add(res)
	}
	metrics.Close()

}

func return200(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
