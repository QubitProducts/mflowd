package main

import (
	"context"
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

func makeSumAggregation(registerer prom.Registerer,
	mInfo *metricInfo) aggregationFn {

	ops := prom.CounterOpts{
		Name: mInfo.name,
		Help: metricHelp(mInfo),
	}
	ctr := prom.NewCounterVec(ops, mInfo.labelNames)
	if err := registerer.Register(ctr); err != nil {
		log.Warnf("Failed to create a counter '%s': %v", mInfo.name, err)
	}

	return func(inmInfo *metricInfo) {
		log.Debugf("[COUNTER AGGR] :: (%s) (%v) <- %f", inmInfo.name,
			inmInfo.labels, inmInfo.value)
		ctr.With(inmInfo.labels).Add(inmInfo.value)
	}
}

func makeGaugeAggregation(registerer prom.Registerer,
	mInfo *metricInfo) aggregationFn {

	luTs := mInfo.timestamp
	ops := prom.GaugeOpts{
		Name: mInfo.name,
		Help: metricHelp(mInfo),
	}
	gm := prom.NewGaugeVec(ops, mInfo.labelNames)
	if err := registerer.Register(gm); err != nil {
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

func makeAggregationFunction(registerer prom.Registerer,
	mInfo *metricInfo) (aggregationFn, error) {

	var err error
	var afn aggregationFn
	switch mInfo.aggrType {
	case "sum":
		afn = makeSumAggregation(registerer, mInfo)
		log.Debugf("New SUM aggregation: %s", mInfo.name)
	case "mean":
		afn = makeGaugeAggregation(registerer, mInfo)
		log.Debugf("New MEAN aggregation: %s", mInfo.name)
	default:
		err = fmt.Errorf("Unknown aggregation type: '%s'", mInfo.aggrType)
	}

	return afn, err
}

type aggregatorContext struct {
	registerer prom.Registerer
	gatherer   prom.Gatherer
	fnTable    map[string]aggregationFn
}

func aggregateMetric(actx *aggregatorContext, mInfo *metricInfo) error {
	afn, ok := actx.fnTable[mInfo.name]
	if !ok {
		var err error
		afn, err = makeAggregationFunction(actx.registerer, mInfo)
		if err != nil {
			return err
		}
		actx.fnTable[mInfo.name] = afn
	}

	afn(mInfo)
	return nil
}

func newAggregatorContext(registry *prom.Registry) *aggregatorContext {
	var ctx aggregatorContext
	if registry == nil {
		ctx.gatherer = prom.DefaultGatherer
		ctx.registerer = prom.DefaultRegisterer
	} else {
		registry = prom.NewRegistry()
		ctx.gatherer = registry
		ctx.registerer = registry
	}

	ctx.fnTable = make(map[string]aggregationFn)
	return &ctx
}

func launchAggregator(ctx context.Context, minfoChan chan *metricInfo,
	pio *promIO) error {

	return launchAggregatorWithCustomRegistry(ctx, minfoChan, pio, nil)
}

func launchAggregatorWithCustomRegistry(ctx context.Context,
	minfoChan chan *metricInfo, pio *promIO, registry *prom.Registry) error {

	log.Debug("Running aggregator ...")
	actx := newAggregatorContext(registry)
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
				msg := promMessage{gatherer: actx.gatherer}
				pio.messageChan <- msg
			}
			break
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
