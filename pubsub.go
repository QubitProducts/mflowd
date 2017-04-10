package main

import (
	"fmt"
	"strings"

	"cloud.google.com/go/pubsub"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
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

func runPubSubPoller(ctx context.Context, source string,
	minfoChan chan *metricInfo) error {

	projectID, subID, err := parseProjectAndSubscriptionIDs(source)
	if err != nil {
		return err
	}

	sub := createPubSubClient(projectID).Subscription(subID)
	cctx, cancel := context.WithCancel(ctx)
	err = sub.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		select {
		case <-ctx.Done():
			cancel()
			msg.Nack()
			return
		default:
			go func() {
				log.Debugf("Got new message: %v", string(msg.Data))
				handleIncomingMessage(minfoChan, msg.Data, msg.PublishTime.Unix())
				msg.Ack() // XXX: should I ack if an error has happened?
			}()
		}
	})

	return err
}
