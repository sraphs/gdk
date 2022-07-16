package redispubsub_test

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"

	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/redispubsub"
)

func ExampleOpenTopic() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	opt, err := redis.ParseURL("redis://localhost:6379")
	if err != nil {
		log.Fatal(err)
	}
	client := redis.NewClient(opt)
	defer client.Close()

	topic, err := redispubsub.OpenTopic(client, "example.my-topic", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}

func ExampleOpenSubscription() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	opt, err := redis.ParseURL("redis://localhost:6379")
	if err != nil {
		log.Fatal(err)
	}
	client := redis.NewClient(opt)
	defer client.Close()

	subscription, err := redispubsub.OpenSubscription(client, "example.my-topic", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}

func Example_openTopicFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/redispubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenTopic creates a *pubsub.Topic from a URL.
	// This URL will Dial the Redis server at the URL in the environment variable
	// REDIS_SERVER_URL and send messages with subject "example.my-topic".
	topic, err := pubsub.OpenTopic(ctx, "redis://example.my-topic")
	if err != nil {
		log.Fatal(err)
	}
	defer topic.Shutdown(ctx)
}

func Example_openSubscriptionFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/pubsub/redispubsub"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// pubsub.OpenSubscription creates a *pubsub.Subscription from a URL.
	// This URL will Dial the Redis server at the URL in the environment variable
	// REDIS_SERVER_URL and receive messages with subject "example.my-topic".
	subscription, err := pubsub.OpenSubscription(ctx, "redis://example.my-topic")
	if err != nil {
		log.Fatal(err)
	}
	defer subscription.Shutdown(ctx)
}
