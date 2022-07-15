package natspubsub_test

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/natspubsub"
)

func ExampleOpenTopic() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	natsConn, err := nats.Connect("nats://nats.example.com")
	if err != nil {
		log.Fatal(err)
	}
	defer natsConn.Close()

	topic, err := natspubsub.OpenTopic(natsConn, "example.mysubject", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}

func ExampleOpenSubscription() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	natsConn, err := nats.Connect("nats://nats.example.com")
	if err != nil {
		log.Fatal(err)
	}
	defer natsConn.Close()

	subscription, err := natspubsub.OpenSubscription(
		natsConn,
		"example.mysubject",
		nil)
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}

func ExampleOpenQueueSubscription() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	natsConn, err := nats.Connect("nats://nats.example.com")
	if err != nil {
		log.Fatal(err)
	}
	defer natsConn.Close()

	subscription, err := natspubsub.OpenSubscription(
		natsConn,
		"example.mysubject",
		&natspubsub.SubscriptionOptions{Queue: "queue1"})
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}

func Example_openTopicFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/natspubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenTopic creates a *pubsub.Topic from a URL.
	// This URL will Dial the NATS server at the URL in the environment variable
	// NATS_SERVER_URL and send messages with subject "example.mysubject".
	topic, err := pubsub.OpenTopic(ctx, "nats://example.mysubject")
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}

func Example_openSubscriptionFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/natspubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenSubscription creates a *pubsub.Subscription from a URL.
	// This URL will Dial the NATS server at the URL in the environment variable
	// NATS_SERVER_URL and receive messages with subject "example.mysubject".
	subscription, err := pubsub.OpenSubscription(ctx, "nats://example.mysubject")
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}

func Example_openQueueSubscriptionFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/natspubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenSubscription creates a *pubsub.Subscription from a URL.
	// This URL will Dial the NATS server at the URL in the environment variable
	// NATS_SERVER_URL and receive messages with subject "example.mysubject"
	// This URL will be parsed and the queue attribute will be used as the Queue parameter when creating the NATS Subscription.
	subscription, err := pubsub.OpenSubscription(ctx, "nats://example.mysubject?queue=myqueue")
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}
