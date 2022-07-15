package mempubsub_test

import (
	"context"
	"log"
	"time"

	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/mempubsub"
)

func ExampleNewSubscription() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// Construct a *pubsub.Topic.
	topic := mempubsub.NewTopic()
	defer topic.Shutdown(ctx)

	// Construct a *pubsub.Subscription for the topic.
	subscription := mempubsub.NewSubscription(topic, 1*time.Minute /* ack deadline */)
	defer subscription.Shutdown(ctx)
}

func ExampleNewTopic() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	topic := mempubsub.NewTopic()
	defer topic.Shutdown(ctx)
}

func Example_openSubscriptionFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/mempubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// Create a topic.
	topic, err := pubsub.OpenTopic(ctx, "mem://topicA")
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)

	// Create a subscription connected to that topic.
	subscription, err := pubsub.OpenSubscription(ctx, "mem://topicA")
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}

func Example_openTopicFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/mempubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	topic, err := pubsub.OpenTopic(ctx, "mem://topicA")
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}
