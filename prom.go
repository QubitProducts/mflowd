package main

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type promMessage struct {
	gatherer prom.Gatherer
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

		promhttp.HandlerFor(pmsg.gatherer, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}

func mflowStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("cogito ergo sum\n"))
}

func exposePrometheusEndpoint(port int, pio *promIO) {
	log.Debugf("Exposing a Prometheus endpoint at %d", port)
	http.HandleFunc("/metrics", mflowPromHandler(pio))
	http.HandleFunc("/status", mflowStatusHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
