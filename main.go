package main

import (
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

func runPoller(sourceType string, source string, minfoChan chan *metricInfo) {
	var err error
	switch sourceType {
	case "pubsub":
		err = runPubSubPoller(source, minfoChan)
		break
	case "file":
		err = runFilePoller(source, minfoChan)
		break
	default:
		log.Errorf("Unknown source type: '%s'", sourceType)
		os.Exit(-1)
	}
	if err != nil {
		log.Errorf("Failed to create poller: %v", err)
		os.Exit(-1)
	}
}

func waitForever() {
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

func run(cmd *cobra.Command, args []string) {
	port, perr := cmd.PersistentFlags().GetInt("port")
	sourceType, serr := cmd.PersistentFlags().GetString("source")
	if perr != nil || serr != nil || port < 0 || sourceType == "" || len(args) != 1 {
		cmd.Usage()
		os.Exit(-1)
	}

	verbose, err := cmd.PersistentFlags().GetBool("verbose")
	if err == nil && verbose {
		log.SetLevel(log.DebugLevel)
	}

	pio := promIO{
		scrapeSignalChan: make(chan bool),
		messageChan:      make(chan promMessage),
	}

	minfoChan := make(chan *metricInfo)
	go exposePrometheusEndpoint("/metrics", port, &pio)
	go runPoller(sourceType, args[0], minfoChan)
	go launchAggregator(minfoChan, &pio)

	waitForever()
	os.Exit(-1)
}

func main() {
	cmd := &cobra.Command{
		Use:   "mflowd [-p port] <-s source-type> <source>",
		Short: "Metrics Flow Prometheus Proxy",
		Run:   run,
	}

	cmd.PersistentFlags().StringP("source", "s", "",
		"Type of metric update event messages source."+
			" Can be either 'punsub' or 'file'")
	cmd.PersistentFlags().IntP("port", "p", 6221,
		"Port to expose for prometheus to scrap the metrics")
	cmd.PersistentFlags().BoolP("verbose", "v", false,
		"Turn on verbose mode")

	cmd.Execute()
}
