package main

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type promMessage struct {
	registry *prom.Registry
}

type promIO struct {
	scrapeSignalChan chan bool
	messageChan      chan promMessage
}

func mflowPromHandler(pio *promIO) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Prometheus wants to scrape some metrics ...")
		pio.scrapeSignalChan <- true
		pmsg := <-pio.messageChan

		promhttp.HandlerFor(pmsg.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}

func makePromIO() *promIO {
	return &promIO{
		scrapeSignalChan: make(chan bool),
		messageChan:      make(chan promMessage),
	}
}

func exposePrometheusEndpoint(endpoint string, port int, pio *promIO) {
	log.Debugf("Exposing a Prometheus endpoint at %d", port)
	http.HandleFunc(endpoint, mflowPromHandler(pio))
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
