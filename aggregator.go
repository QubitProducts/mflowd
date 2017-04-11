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

type aggregation struct {
	afn   aggregationFn
	atype string
}

func metricHelp(minfo *metricInfo) string {
	return "void"
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

func makeAggregation(registerer prom.Registerer,
	mInfo *metricInfo) (*aggregation, error) {

	var err error
	var afn aggregationFn
	switch mInfo.aggrType {
	case "counter":
		afn = makeSumAggregation(registerer, mInfo)
		log.Debugf("New COUNTER aggregation: %s", mInfo.name)
	case "gauge":
		afn = makeGaugeAggregation(registerer, mInfo)
		log.Debugf("New GAUGE aggregation: %s", mInfo.name)
	default:
		err = fmt.Errorf("Unknown aggregation type: '%s'", mInfo.aggrType)
	}
	if err != nil {
		return nil, err
	}

	return &aggregation{
		afn:   afn,
		atype: mInfo.aggrType,
	}, nil
}

type aggregatorContext struct {
	registerer prom.Registerer
	gatherer   prom.Gatherer
	aggrTable  map[string]*aggregation
}

func aggregateMetric(actx *aggregatorContext, mInfo *metricInfo) error {
	aggr, ok := actx.aggrTable[mInfo.name]
	if !ok {
		var err error
		aggr, err = makeAggregation(actx.registerer, mInfo)
		if err != nil {
			return err
		}

		actx.aggrTable[mInfo.name] = aggr
	}
	if aggr.atype != mInfo.aggrType {
		return fmt.Errorf("Can't aggregate metric \"%s\" type \"%s\" as it is"+
			" already regisited with different type \"%s\"", mInfo.name,
			mInfo.aggrType, aggr.atype)
	}

	aggr.afn(mInfo)
	return nil
}

func newAggregatorContext(registry *prom.Registry) *aggregatorContext {
	var ctx aggregatorContext
	if registry == nil {
		ctx.gatherer = prom.DefaultGatherer
		ctx.registerer = prom.DefaultRegisterer
	} else {
		ctx.gatherer = registry
		ctx.registerer = registry
	}

	ctx.aggrTable = make(map[string]*aggregation)
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
