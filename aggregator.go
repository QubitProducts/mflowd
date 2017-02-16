package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	prom "github.com/prometheus/client_golang/prometheus"
)

type metricInfo struct {
	name       string
	aggrType   string
	value      float64
	labelNames []string
	labels     prom.Labels
	timestamp  int64
}

type aggregationFn func(*metricInfo)

func metricHelp(minfo *metricInfo) string {
	return "some-shit-here"
}

func makeSumAggregation(registry *prom.Registry,
	mInfo *metricInfo) aggregationFn {

	ops := prom.CounterOpts{
		Name: mInfo.name,
		Help: metricHelp(mInfo),
	}
	ctr := prom.NewCounterVec(ops, mInfo.labelNames)
	if err := registry.Register(ctr); err != nil {
		log.Warnf("Failed to create a counter '%s': %v", mInfo.name, err)
	}

	return func(inmInfo *metricInfo) {
		log.Debugf("[COUNTER AGGR] :: (%s) (%v) <- %f", inmInfo.name,
			inmInfo.labels, inmInfo.value)
		ctr.With(inmInfo.labels).Add(inmInfo.value)
	}
}

func makeGaugeAggregation(registry *prom.Registry,
	mInfo *metricInfo) aggregationFn {

	luTs := mInfo.timestamp
	ops := prom.GaugeOpts{
		Name: mInfo.name,
		Help: metricHelp(mInfo),
	}
	gm := prom.NewGaugeVec(ops, mInfo.labelNames)
	if err := registry.Register(gm); err != nil {
		log.Warnf("Failed to create a gauge '%s': %v", mInfo.name, err)
	}

	return func(inmInfo *metricInfo) {
		if inmInfo.timestamp >= luTs {
			log.Debugf("[GAUGE AGGR] :: (%s) (%v) <- %f", inmInfo.name,
				inmInfo.labels, inmInfo.value)
			gm.With(inmInfo.labels).Set(inmInfo.value)
			luTs = inmInfo.timestamp
		}
	}
}

func makeAggregationFunction(registry *prom.Registry,
	mInfo *metricInfo) (aggregationFn, error) {

	var err error
	var afn aggregationFn
	switch mInfo.aggrType {
	case "sum":
		afn = makeSumAggregation(registry, mInfo)
		log.Debugf("New SUM aggregation: %s", mInfo.name)
	case "mean":
		afn = makeGaugeAggregation(registry, mInfo)
		log.Debugf("New MEAN aggregation: %s", mInfo.name)
	default:
		err = fmt.Errorf("Unknown aggregation type: '%s'", mInfo.aggrType)
	}

	return afn, err
}

type aggregatorContext struct {
	registry *prom.Registry
	fnTable  map[string]aggregationFn
}

func aggregateMetric(actx *aggregatorContext, mInfo *metricInfo) error {
	afn, ok := actx.fnTable[mInfo.name]
	if !ok {
		var err error
		afn, err = makeAggregationFunction(actx.registry, mInfo)
		if err != nil {
			return err
		}
		actx.fnTable[mInfo.name] = afn
	}

	afn(mInfo)
	return nil
}

func newContext() *aggregatorContext {
	return &aggregatorContext{
		registry: prom.NewRegistry(),
		fnTable:  make(map[string]aggregationFn),
	}
}

func launchAggregator(minfoChan chan *metricInfo, pio *promIO) {
	log.Debug("Running aggregator ...")
	actx := newContext()
	for {
		select {
		case mInfo, chanOk := <-minfoChan:
			if chanOk {
				err := aggregateMetric(actx, mInfo)
				if err != nil {
					log.Warnf("Failed to aggregate metric '%v': %v",
						mInfo.name, err)
				}
			}

			break
		case _, chanOk := <-pio.scrapeSignalChan:
			if chanOk {
				log.Debug("Sending aggregated metrics ...")
				msg := promMessage{registry: actx.registry}
				actx = newContext()
				pio.messageChan <- msg
			}
			break
		}
	}
}
