package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/sync/errgroup"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

func runPoller(ctx context.Context, sourceType string,
	source string, minfoChan chan *metricInfo) error {

	var err error
	switch sourceType {
	case "pubsub":
		err = runPubSubPoller(ctx, source, minfoChan)
		break
	case "file":
		err = runFilePoller(ctx, source, minfoChan)
		break
	default:
		err = fmt.Errorf("Unknown source type: '%s'", sourceType)
	}

	return err
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

	ctx := context.Background()
	minfoChan := make(chan *metricInfo)

	var g errgroup.Group
	go exposePrometheusEndpoint("/metrics", port, &pio)
	g.Go(runPoller(ctx, sourceType, args[0], minfoChan))
	g.Go(launchAggregator(ctx, minfoChan, &pio))

	if err = g.Wait(); err != nil {
		log.Errorf("Error: %v", err)
	}
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
