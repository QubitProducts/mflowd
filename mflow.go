package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	prom "github.com/prometheus/client_golang/prometheus"
)

type labelNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type metricUpdateEvent struct {
	Name                string           `json:"name"`
	LabelNameValuePairs []labelNameValue `json:"labelNameValuePairs"`
	Value               float64          `json:"value"`
}

// sort.Interface
type byLabelName []labelNameValue

func (a byLabelName) Len() int           { return len(a) }
func (a byLabelName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byLabelName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func sortedLabelNames(labelNameValuePairs []labelNameValue) []string {
	sort.Sort(byLabelName(labelNameValuePairs))
	lnames := make([]string, len(labelNameValuePairs))
	for i, nvPair := range labelNameValuePairs {
		lnames[i] = nvPair.Name
	}

	return lnames
}

func toPrometheusLabels(labelNameValuePairs []labelNameValue) prom.Labels {
	labels := make(map[string]string)
	for _, nvp := range labelNameValuePairs {
		labels[nvp.Name] = nvp.Value
	}

	return labels
}

func toMetricInfo(event *metricUpdateEvent, ts int64) (*metricInfo, error) {
	nameItems := strings.Split(event.Name, "_")
	if len(nameItems) != 2 {
		return nil, fmt.Errorf("Metric %s does not have type suffix", event.Name)
	}

	return &metricInfo{
		name:       nameItems[0],
		aggrType:   nameItems[1],
		value:      event.Value,
		labelNames: sortedLabelNames(event.LabelNameValuePairs),
		labels:     toPrometheusLabels(event.LabelNameValuePairs),
		timestamp:  ts,
	}, nil
}

func handleIncomingMessage(minfoChan chan *metricInfo,
	msgData []byte, msgTs int64) {
	var event metricUpdateEvent
	if err := json.Unmarshal(msgData, &event); err != nil {
		log.Warnf("Failed to unmarshal message: %v (orig: %v)", err, string(msgData))
	} else {
		minfo, err := toMetricInfo(&event, msgTs)
		if err == nil {
			minfoChan <- minfo
		} else {
			log.Warnf("Failed to parse metric update event: %v (orig: %v)",
				err, string(msgData))
		}
	}
}
