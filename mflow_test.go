package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventToMetricInfo_badName(t *testing.T) {
	event := makeMetricUpdateEvent("eventWithNoType", 1.0)
	_, err := toMetricInfo(event, time.Now().Unix())
	if err == nil {
		t.Log("Metric names with no type suffix should not be allowed")
		t.Fail()
	}
}

func TestEventToMetricInfo_nameWithManyDashes(t *testing.T) {
	event := makeMetricUpdateEvent("event_with_many_dashes_in_name_and_type", 1.0)
	minfo, err := toMetricInfo(event, time.Now().Unix())
	if err != nil {
		t.Errorf("toMetricInfo: %v", err)
	}

	assert.Equal(t, "event_with_many_dashes_in_name_and", minfo.name)
	assert.Equal(t, "type", minfo.aggrType)
	assert.Equal(t, 1.0, minfo.value)
}

func TestEventToMetricInfo_withLabels(t *testing.T) {
	labels := []string{"l3:v3", "l1:v1", "l2:v2"}
	event := makeMetricUpdateEventWithLabels("some_name_counter", 123.0, labels)
	minfo, err := toMetricInfo(event, time.Now().Unix())
	if err != nil {
		t.Errorf("toMetricInfo: %v", err)
	}

	assert.Equal(t, "some_name", minfo.name)
	assert.Equal(t, "counter", minfo.aggrType)
	assert.Equal(t, 123.0, minfo.value)

	assert.Equal(t, []string{"l1", "l2", "l3"}, minfo.labelNames)
	assert.Contains(t, minfo.labels, "l1", "l2", "l3")

	assert.Equal(t, "v1", minfo.labels["l1"])
	assert.Equal(t, "v2", minfo.labels["l2"])
	assert.Equal(t, "v3", minfo.labels["l3"])
}

func makeMetricUpdateEvent(name string, value float64) *metricUpdateEvent {
	return makeMetricUpdateEventWithLabels(name, value, []string{})
}

func makeMetricUpdateEventWithLabels(name string, value float64,
	labels []string) *metricUpdateEvent {

	var lvp []labelNameValue
	for _, l := range labels {
		items := strings.Split(l, ":")
		lvp = append(lvp, labelNameValue{
			Name:  items[0],
			Value: items[1],
		})
	}

	return &metricUpdateEvent{
		Name:                name,
		Value:               value,
		LabelNameValuePairs: lvp,
	}
}
