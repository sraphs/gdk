package kafkapubsub_test

import (
	"context"
	"log"

	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/kafkapubsub"
)

func ExampleOpenTopic() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// The set of brokers in the Kafka cluster.
	addrs := []string{"1.2.3.4:9092"}
	// The Kafka client configuration to use.
	config := kafkapubsub.MinimalConfig()

	// Construct a *pubsub.Topic.
	topic, err := kafkapubsub.OpenTopic(addrs, config, "my-topic", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}

func ExampleOpenSubscription() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// The set of brokers in the Kafka cluster.
	addrs := []string{"1.2.3.4:9092"}
	// The Kafka client configuration to use.
	config := kafkapubsub.MinimalConfig()

	// Construct a *pubsub.Subscription, joining the consumer group "my-group"
	// and receiving messages from "my-topic".
	subscription, err := kafkapubsub.OpenSubscription(
		addrs, config, "my-group", []string{"my-topic"}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}

func Example_openTopicFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/kafkapubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenTopic creates a *pubsub.Topic from a URL.
	// The host + path are the topic name to send to.
	// The set of brokers must be in an environment variable KAFKA_BROKERS.
	topic, err := pubsub.OpenTopic(ctx, "kafka://my-topic")
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}

func Example_openSubscriptionFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/kafkapubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenSubscription creates a *pubsub.Subscription from a URL.
	// The host + path are used as the consumer group name.
	// The "topic" query parameter sets one or more topics to subscribe to.
	// The set of brokers must be in an environment variable KAFKA_BROKERS.
	subscription, err := pubsub.OpenSubscription(ctx,
		"kafka://my-group?topic=my-topic")
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}
