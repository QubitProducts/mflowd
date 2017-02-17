package main

import (
	"bufio"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

func runFilePoller(filepath string, minfoChan chan *metricInfo) error {
	log.Debugf("Reading metric update events from file: %s", filepath)
	f, err := os.Open(filepath)
	if err != nil {
		return err
	}

	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		go handleIncomingMessage(minfoChan, []byte(s.Text()), time.Now().Unix())
	}

	log.Debugf("No more metric update events in '%s'", filepath)
	return nil
}
