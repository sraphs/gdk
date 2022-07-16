package pulsarpubsub_test

import (
	"context"
	"log"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/pulsarpubsub"
)

func ExampleOpenTopic() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	localPulsarURL := "pulsar://localhost:6650"
	config := pulsarpubsub.MinimalConfig(localPulsarURL)
	client, err := pulsar.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Construct a *pubsub.Topic.
	topic, err := pulsarpubsub.OpenTopic(client, &pulsarpubsub.TopicOptions{
		ProducerOptions: pulsar.ProducerOptions{
			Topic: "my-topic",
		},
		KeyName: "",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}

func ExampleOpenSubscription() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	localPulsarURL := "pulsar://localhost:6650"
	config := pulsarpubsub.MinimalConfig(localPulsarURL)
	client, err := pulsar.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	// Construct a *pubsub.Subscription, use the SubscriptionName "my-sub"
	// and receiving messages from "my-topic".
	subscription, err := pulsarpubsub.OpenSubscription(client, &pulsarpubsub.SubscriptionOptions{
		ConsumerOptions: pulsar.ConsumerOptions{
			Topic:            "my-topic",
			SubscriptionName: "my-sub",
		},
		KeyName: "",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}

func Example_openTopicFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/pulsarpubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenTopic creates a *pubsub.Topic from a URL.
	// The host + path are the topic name to send to.
	// The set of brokers must be in an environment variable KAFKA_BROKERS.
	topic, err := pubsub.OpenTopic(ctx, "pulsar://my-topic")
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}

func Example_openSubscriptionFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/pulsarpubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenSubscription creates a *pubsub.Subscription from a URL.
	// The host + path are used as the consumer group name.
	// The "topic" query parameter sets one or more topics to subscribe to.
	// The set of brokers must be in an environment variable KAFKA_BROKERS.
	subscription, err := pubsub.OpenSubscription(ctx,
		"pulsar://my-sub?topic=my-topic")
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}
