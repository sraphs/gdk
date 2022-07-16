---
title: "Subscribe to Messages from a Topic"
date: 2019-03-26T09:44:33-07:00
lastmod: 2019-07-29T12:00:00-07:00
weight: 2
toc: true
---

Subscribing to receive message from a topic with the GDK takes three steps:

1. [Open a subscription][] to a topic with the Pub/Sub service of your choice (once per
   subscription).
2. [Receive and acknowledge messages][] from the topic. After completing any
   work related to the message, use the Ack method to prevent it from being
   redelivered.

[Open a subscription]: {{< ref "#opening" >}}
[Receive and acknowledge messages]: {{< ref "#receiving" >}}

<!--more-->

## Opening a Subscription {#opening}

The first step in subscribing to receive messages from a topic is
to instantiate a portable [`*pubsub.Subscription`][] for your service.

The easiest way to do so is to use [`pubsub.OpenSubscription`][]
and a service-specific URL pointing to the topic, making sure you
["blank import"][] the driver package to link it in.

```go
import (
    "context"

    "github.com/sraphs/gdk/pubsub"
    _ "github.com/sraphs/gdk/pubsub/<driver>"
)
...
ctx := context.Background()
subs, err := pubsub.OpenSubscription(ctx, "<driver-url>")
if err != nil {
    return fmt.Errorf("could not open topic subscription: %v", err)
}
defer subs.Shutdown(ctx)
// subs is a *pubsub.Subscription; see usage below
...
```

See [Concepts: URLs][] for general background and the [guide below][]
for URL usage for each supported service.

Alternatively, if you need fine-grained
control over the connection settings, you can call the constructor function in
the driver package directly (like `gcppubsub.OpenSubscription`).

```go
import "github.com/sraphs/gdk/pubsub/<driver>"
...
subs, err := <driver>.OpenSubscription(...)
...
```

You may find the [`wire` package][] useful for managing your initialization code
when switching between different backing services.

See the [guide below][] for constructor usage for each supported service.

[guide below]: {{< ref "#services" >}}
[`pubsub.OpenSubscription`]:
https://godoc.org/github.com/sraphs/gdk/pubsub#OpenTopic
["blank import"]: https://golang.org/doc/effective_go.html#blank_import
[Concepts: URLs]: {{< ref "/concepts/urls.md" >}}
[`wire` package]: http://github.com/google/wire

## Receiving and Acknowledging Messages {#receiving}

A simple subscriber that operates on
[messages](https://godoc.org/github.com/sraphs/gdk/pubsub#Message) serially looks like
this:

{{< goexample src="github.com/sraphs/gdk/pubsub.ExampleSubscription_Receive" imports="0" >}}

If you want your subscriber to operate on incoming messages concurrently,
you can start multiple goroutines:

{{< goexample src="github.com/sraphs/gdk/pubsub.ExampleSubscription_Receive_concurrent" imports="0" >}}

Note that the [semantics of message delivery][] can vary by backing service.

[`*pubsub.Subscription`]: https://godoc.org/github.com/sraphs/gdk/pubsub#Subscription
[semantics of message delivery]: https://godoc.org/github.com/sraphs/gdk/pubsub#hdr-At_most_once_and_At_least_once_Delivery

## Other Usage Samples

* [CLI Sample](https://github.com/sraphs/gdk/tree/master/samples/gocdk-pubsub)
* [Order Processor sample](https://github.com/sraphs/gdk/tutorials/order/)
* [pubsub package examples](https://godoc.org/github.com/sraphs/gdk/pubsub#pkg-examples)

## Supported Pub/Sub Services {#services}

### RabbitMQ {#rabbitmq}

The GDK can receive messages from an [AMQP 0.9.1][] queue, the dialect of
AMQP spoken by [RabbitMQ][]. A RabbitMQ URL only includes the queue name.
The RabbitMQ's server is discovered from the `RABBIT_SERVER_URL` environment
variable (which is something like `amqp://guest:guest@localhost:5672/`).

{{< goexample "github.com/sraphs/gdk/pubsub/rabbitpubsub.Example_openSubscriptionFromURL" >}}

[AMQP 0.9.1]: https://www.rabbitmq.com/protocol.html
[RabbitMQ]: https://www.rabbitmq.com

#### RabbitMQ Constructor {#rabbitmq-ctor}

The [`rabbitpubsub.OpenSubscription`][] constructor opens a RabbitMQ queue.
You must first create an [`*amqp.Connection`][] to your RabbitMQ instance.

{{< goexample "github.com/sraphs/gdk/pubsub/rabbitpubsub.ExampleOpenSubscription" >}}

[`*amqp.Connection`]: https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Connection
[`rabbitpubsub.OpenSubscription`]: https://godoc.org/github.com/sraphs/gdk/pubsub/rabbitpubsub#OpenSubscription

### NATS {#nats}

The GDK can publish to a [NATS][] subject. A NATS URL only includes the
subject name. The NATS server is discovered from the `NATS_SERVER_URL`
environment variable (which is something like `nats://nats.example.com`).

{{< goexample "github.com/sraphs/gdk/pubsub/natspubsub.Example_openSubscriptionFromURL" >}}

NATS guarantees at-most-once delivery; it will never redeliver a message.
Therefore, `Message.Ack` is a no-op.

To parse messages [published via the GDK][publish#nats], the NATS driver
will first attempt to decode the payload using [gob][]. Failing that, it will
return the message payload as the `Data` with no metadata to accomodate
subscribing to messages coming from a source not using the GDK.

[gob]: https://golang.org/pkg/encoding/gob/
[NATS]: https://nats.io/
[publish#nats]: {{< ref "./publish.md#nats" >}}

#### NATS Constructor {#nats-ctor}

The [`natspubsub.OpenSubscription`][] constructor opens a NATS subject as a
topic. You must first create an [`*nats.Conn`][] to your NATS instance.

{{< goexample "github.com/sraphs/gdk/pubsub/natspubsub.ExampleOpenSubscription" >}}

[`*nats.Conn`]: https://godoc.org/github.com/nats-io/go-nats#Conn
[`natspubsub.OpenSubscription`]: https://godoc.org/github.com/sraphs/gdk/pubsub/natspubsub#OpenSubscription

### Kafka {#kafka}

The GDK can receive messages from a [Kafka][] cluster.
A Kafka URL includes the consumer group name, plus at least one instance
of a query parameter specifying the topic to subscribe to.
The brokers in the Kafka cluster are discovered from the
`KAFKA_BROKERS` environment variable (which is a comma-delimited list of
hosts, something like `1.2.3.4:9092,5.6.7.8:9092`).

{{< goexample "github.com/sraphs/gdk/pubsub/kafkapubsub.Example_openSubscriptionFromURL" >}}

[Kafka]: https://kafka.apache.org/

#### Kafka Constructor {#kafka-ctor}

The [`kafkapubsub.OpenSubscription`][] constructor creates a consumer in a
consumer group, subscribed to one or more topics.

In addition to the list of brokers, you'll need a [`*sarama.Config`][], which
exposes many knobs that can affect performance and semantics; review and set
them carefully. [`kafkapubsub.MinimalConfig`][] provides a minimal config to
get you started.

{{< goexample "github.com/sraphs/gdk/pubsub/kafkapubsub.ExampleOpenSubscription" >}}

[`*sarama.Config`]: https://godoc.org/github.com/Shopify/sarama#Config
[`kafkapubsub.OpenSubscription`]: https://godoc.org/github.com/sraphs/gdk/pubsub/kafkapubsub#OpenSubscription
[`kafkapubsub.MinimalConfig`]: https://godoc.org/github.com/sraphs/gdk/pubsub/kafkapubsub#MinimalConfig

### In-Memory {#mem}

The GDK includes an in-memory Pub/Sub provider useful for local testing.
The names in `mem://` URLs are a process-wide namespace, so subscriptions to
the same name will receive messages posted to that topic. For instance, if
you open a topic `mem://topicA` and open two subscriptions with
`mem://topicA`, you will have two subscriptions to the same topic.

{{< goexample "github.com/sraphs/gdk/pubsub/mempubsub.Example_openSubscriptionFromURL" >}}

#### In-Memory Constructor {#mem-ctor}

To create a subscription to an in-memory Pub/Sub topic, pass the [topic you
created][publish-mem-ctor] into the [`mempubsub.NewSubscription` function][].
You will also need to pass an acknowledgement deadline: once a message is
received, if it is not acknowledged after the deadline elapses, then it will be
redelivered.

{{< goexample "github.com/sraphs/gdk/pubsub/mempubsub.ExampleNewSubscription" >}}

[`mempubsub.NewSubscription` function]: https://godoc.org/github.com/sraphs/gdk/pubsub/mempubsub#NewSubscription
[publish-mem-ctor]: {{< ref "./publish.md#mem-ctor" >}}



### Redis {#redis}

The GDK can publish to a [Redis][] subject. A Redis URL only includes the
subject name. The Redis server is discovered from the `REDIS_SERVER_URL`
environment variable (which is something like `redis://redis.example.com`).

{{< goexample "github.com/sraphs/gdk/pubsub/redispubsub.Example_openSubscriptionFromURL" >}}

Because Redis does not natively support metadata, messages sent to Redis will
be encoded with [gob][].

[gob]: https://golang.org/pkg/encoding/gob/
[Redis]: https://redis.io/

#### Redis Constructor {#redis-ctor}

The [`redispubsub.OpenSubscription`][] constructor opens a Redis subject as a topic. You
must first create an [`*redis.Client`][] to your Redis instance.

{{< goexample "github.com/sraphs/gdk/pubsub/redispubsub.ExampleOpenSubscription" >}}

[`*redis.Client`]: https://godoc.org/github.com/go-redis/redis/v8#Client
[`redispubsub.OpenSubscription`]: https://godoc.org/github.com/sraphs/gdk/pubsub/redispubsub#OpenSubscription


### Pulsar {#pulsar}

The GDK can publish to a [Pulsar][] subject. A Pulsar URL only includes the
subject name. The Pulsar server is discovered from the `Pulsar_SERVER_URL`
environment variable (which is something like `pulsar://localhost:6650`).

{{< goexample "github.com/sraphs/gdk/pubsub/pulsarpubsub.Example_openSubscriptionFromURL" >}}

[Pulsar]: https://pulsar.apache.org/

#### Pulsar Constructor {#pulsar-ctor}

The [`pulsarpubsub.OpenSubscription`][] constructor opens a Pulsar topic to publish
messages to.

{{< goexample "github.com/sraphs/gdk/pubsub/pulsarpubsub.ExampleOpenSubscription" >}}

[`pulsarpubsub.OpenSubscription`]: https://godoc.org/github.com/sraphs/gdk/pubsub/pulsarpubsub#OpenSubscription
[`pulsarpubsub.MinimalConfig`]: https://godoc.org/github.com/sraphs/gdk/pubsub/pulsarpubsub#MinimalConfig