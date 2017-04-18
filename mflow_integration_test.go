package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const testPort = 9889
const testFile = "misc/test_data.txt"

func TestAllComponentsWorkTogether(t *testing.T) {
	go runTheDaemon(testFile, "file", testPort)

	expectedMetrics := buildExpectedMetricsMap()

	url := fmt.Sprintf("http://localhost:%d/metrics", testPort)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Logf("Faield to construct request for %v", url)
		t.Fail()
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Faield to GET metrics from %s", url)
		t.Fail()
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		l := strings.Trim(scanner.Text(), " \n")
		_, exists := expectedMetrics[l]
		if exists {
			expectedMetrics[l] = true
		}
	}

	for k, v := range expectedMetrics {
		if !v {
			t.Logf("Expected metric [%s] not found in the registry", k)
			t.Fail()
		}
	}
}

func buildExpectedMetricsMap() map[string]bool {
	// expected metrics values after aggregating
	// data from `testFile`
	m := make(map[string]bool)
	m["numApiCalls{userId=\"user1\"} 114"] = false
	m["avgNumEvents{userId=\"user1\"} 1"] = false
	m["avgNumEvents{userId=\"user2\"} 2.875"] = false
	m["avgLatency{userId=\"user1\"} 666638"] = false
	return m
}
