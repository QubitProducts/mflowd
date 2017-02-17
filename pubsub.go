package main

import (
	"context"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"google.golang.org/cloud/pubsub"
)

func createPubSubClient(projectID string) *pubsub.Client {
	client, err := pubsub.NewClient(context.Background(), projectID)
	if err != nil {
		log.Fatal("Failed to create Pub/Sub client", err)
	}

	return client
}

func parseProjectAndSubscriptionIDs(source string) (string, string, error) {
	items := strings.Split(source, "/")
	if len(items) != 2 {
		return "", "", fmt.Errorf("Can not parse pubsub source: %s", source)
	}

	return items[0], items[1], nil
}

func runPubSubPoller(source string, minfoChan chan *metricInfo) error {
	projectID, subID, err := parseProjectAndSubscriptionIDs(source)
	if err != nil {
		return err
	}

	sub := createPubSubClient(projectID).Subscription(subID)
	it, err := sub.Pull(context.Background())
	if err != nil {
		return err
	}

	defer it.Stop()
	for {
		msg, err := it.Next()
		if err != nil {
			// should we handle it in a more graceful way?
			log.Fatalf("Failed to read next pub/sub message: %v", err)
		}
		go func() {
			log.Debugf("Got new message: %v", string(msg.Data))
			handleIncomingMessage(minfoChan, msg.Data, msg.PublishTime.Unix())
			msg.Done(true) // XXX: should I ack if an error has happened?
		}()
	}
}
