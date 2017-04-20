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
	// Usually subscription URI looks like this:
	// projects/<projectName>/subscriptions/<subscriptionName>
	if len(items) != 4 {
		return "", "", fmt.Errorf("Can not parse pubsub source: %s", source)
	}

	return items[1], items[3], nil
}

func runPubSubPoller(ctx context.Context, source string,
	minfoChan chan *metricInfo) error {
	cctx, cancel := context.WithCancel(ctx)

	projectID, subID, err := parseProjectAndSubscriptionIDs(source)
	if err != nil {
		cancel()
		return err
	}

	log.Debugf("Polling a pub/sub subscription %s of project %s ...",
		subID, projectID)

	sub := createPubSubClient(projectID).Subscription(subID)
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
