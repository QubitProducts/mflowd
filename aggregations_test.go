package main

import (
	"fmt"
	"testing"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestCounterAggregation(t *testing.T) {
	registry := prom.NewRegistry()
	actx := newAggregatorContext(registry)

	firstMinfo := makeMetricInfo("sum_metric", "counter", 1.0)
	doAggregateMetric(t, actx, firstMinfo)
	secondMinfo := makeMetricInfo("sum_metric", "counter", 2.0)
	doAggregateMetric(t, actx, secondMinfo)

	resMetrics, err := registry.Gather()
	if err != nil {
		t.Errorf("Gather() failed: %v", err)
	}
	assert.Equal(t, 1, len(resMetrics), "there should be one metric available")

	m := resMetrics[0]
	assert.Equal(t, "sum_metric", *m.Name)
	assert.Equal(t, dto.MetricType_COUNTER, *m.Type)
	assert.Equal(t, 1, len(m.Metric))
	assert.Equal(t, 3.0, *m.Metric[0].Counter.Value)
}

func TestGaugeAggregation(t *testing.T) {
	registry := prom.NewRegistry()
	actx := newAggregatorContext(registry)

	firstMinfo := makeMetricInfo("gauge_metric", "gauge", 1.0)
	doAggregateMetric(t, actx, firstMinfo)
	secondMinfo := makeMetricInfo("gauge_metric", "gauge", 100.0)
	doAggregateMetric(t, actx, secondMinfo)

	resMetrics, err := registry.Gather()
	if err != nil {
		t.Errorf("Gather() failed: %v", err)
	}

	assert.Equal(t, 1, len(resMetrics), "there should be one metric available")
	m := resMetrics[0]

	assert.Equal(t, "gauge_metric", *m.Name)
	assert.Equal(t, dto.MetricType_GAUGE, *m.Type)
	assert.Equal(t, 1, len(m.Metric))
	assert.Equal(t, 100.0, *m.Metric[0].Gauge.Value)

	// now make sure that the gague aggregation won't be overwritten if
	// a timestamp of a new metric is older than the one of the aggregation
	thirdMinfo := makeMetricInfoWithTs("gauge_metric", "gauge", 100500.0,
		time.Now().Unix()-100500)
	doAggregateMetric(t, actx, thirdMinfo)

	resMetrics, err = registry.Gather()
	if err != nil {
		t.Errorf("Gather() failed: %v", err)
	}

	m = resMetrics[0]
	assert.Equal(t, 100.0, *m.Metric[0].Gauge.Value)
}

func TestAggregationLabels(t *testing.T) {
	registry := prom.NewRegistry()
	actx := newAggregatorContext(registry)

	firstMinfo := makeMetricInfo("sum_metric", "counter", 2.0)
	firstMinfo.labelNames = []string{"one", "two", "three"}
	firstMinfo.labels["one"] = "1"
	firstMinfo.labels["two"] = "2"
	firstMinfo.labels["three"] = "3"
	doAggregateMetric(t, actx, firstMinfo)

	secondMinfo := makeMetricInfo("sum_metric", "counter", 4.0)
	secondMinfo.labelNames = []string{"one", "two", "three"}
	secondMinfo.labels["one"] = "3"
	secondMinfo.labels["two"] = "2"
	secondMinfo.labels["three"] = "1"
	doAggregateMetric(t, actx, secondMinfo)

	resMetrics, err := registry.Gather()
	if err != nil {
		t.Errorf("Gather() failed: %v", err)
	}

	assert.Equal(t, 1, len(resMetrics))
	m := resMetrics[0]
	assert.Equal(t, 2, len(m.Metric))
	assert.Equal(t, 2.0, *m.Metric[0].Counter.Value)
	assert.Equal(t, 4.0, *m.Metric[1].Counter.Value)

	assertThatLabelsAreEqual(t, []string{"one:1", "two:2", "three:3"},
		m.Metric[0].Label)
	assertThatLabelsAreEqual(t, []string{"one:3", "two:2", "three:1"},
		m.Metric[1].Label)
}

func TestAggregationSameNameDifferentMetricTypes(t *testing.T) {
	registry := prom.NewRegistry()
	actx := newAggregatorContext(registry)

	firstMinfo := makeMetricInfo("same_name", "counter", 1.0)
	doAggregateMetric(t, actx, firstMinfo)
	secondMinfo := makeMetricInfo("same_name", "gauge", 2.0)
	err := aggregateMetric(actx, secondMinfo)
	if err == nil {
		t.Log("It should not be possible to aggregate two metrics with the same" +
			" name but different types")
		t.Fail()
	}

	resMetrics, err := registry.Gather()
	if err != nil {
		t.Errorf("Gather() failed: %v", err)
	}

	assert.Equal(t, 1, len(resMetrics))
	m := resMetrics[0]
	assert.Equal(t, 1, len(m.Metric))
	assert.Equal(t, dto.MetricType_COUNTER, *m.Type)
	assert.Equal(t, 1.0, *m.Metric[0].Counter.Value)
}

func TestAggregationWithUnknownType(t *testing.T) {
	registry := prom.NewRegistry()
	actx := newAggregatorContext(registry)

	minfo := makeMetricInfo("same_name", "some_unknown_type", 1.0)
	err := aggregateMetric(actx, minfo)
	if err == nil {
		t.Log("It should not be possible to aggregate metric with unknown type")
		t.Fail()
	}
}

func makeMetricInfo(name string, aggrType string, value float64) *metricInfo {
	return makeMetricInfoWithTs(name, aggrType, value, time.Now().Unix())
}

func makeMetricInfoWithTs(name string, aggrType string,
	value float64, ts int64) *metricInfo {

	return &metricInfo{
		name:      name,
		aggrType:  aggrType,
		value:     value,
		timestamp: ts,
		labels:    make(map[string]string),
	}
}

func doAggregateMetric(t *testing.T, actx *aggregatorContext,
	mInfo *metricInfo) {

	err := aggregateMetric(actx, mInfo)
	if err != nil {
		t.Errorf("aggregateMetric: %v", err)
	}
}

func assertThatLabelsAreEqual(t *testing.T, expected []string,
	labels []*dto.LabelPair) {

	assert.Equal(t, len(expected), len(labels))
	for _, pair := range labels {
		l := fmt.Sprintf("%s:%s", *pair.Name, *pair.Value)

		found := false
		for _, el := range expected {
			if el == l {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Unexpected label value: \"%v\"", l)
			t.Fail()
		}
	}
}
