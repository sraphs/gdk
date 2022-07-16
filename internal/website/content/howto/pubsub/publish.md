---
title: "Publish Messages to a Topic"
date: 2019-03-26T09:44:15-07:00
lastmod: 2019-07-29T12:00:00-07:00
weight: 1
toc: true
---

Publishing a message to a topic with the GDK takes two steps:

1. [Open a topic][] with the Pub/Sub provider of your choice (once per topic).
2. [Send messages][] on the topic.

[Open a topic]: {{< ref "#opening" >}}
[Send messages]: {{< ref "#sending" >}}

<!--more-->

## Opening a Topic {#opening}

The first step in publishing messages to a topic is to instantiate a
portable [`*pubsub.Topic`][] for your service.

The easiest way to do so is to use [`pubsub.OpenTopic`][] and a service-specific URL
pointing to the topic, making sure you ["blank import"][] the driver package to
link it in.

```go
import (
    "context"

    "github.com/sraphs/gdk/pubsub"
    _ "github.com/sraphs/gdk/pubsub/<driver>"
)
...
ctx := context.Background()
topic, err := pubsub.OpenTopic(ctx, "<driver-url>")
if err != nil {
    return fmt.Errorf("could not open topic: %v", err)
}
defer topic.Shutdown(ctx)
// topic is a *pubsub.Topic; see usage below
...
```

See [Concepts: URLs][] for general background and the [guide below][]
for URL usage for each supported service.

Alternatively, if you need fine-grained
control over the connection settings, you can call the constructor function in
the driver package directly (like `gcppubsub.OpenTopic`).

```go
import "github.com/sraphs/gdk/pubsub/<driver>"
...
topic, err := <driver>.OpenTopic(...)
...
```

You may find the [`wire` package][] useful for managing your initialization code
when switching between different backing services.

See the [guide below][] for constructor usage for each supported service.

[guide below]: {{< ref "#services" >}}
[`*pubsub.Topic`]: https://godoc.org/github.com/sraphs/gdk/pubsub#Topic
[`pubsub.OpenTopic`]:
https://godoc.org/github.com/sraphs/gdk/pubsub#OpenTopic
["blank import"]: https://golang.org/doc/effective_go.html#blank_import
[Concepts: URLs]: {{< ref "/concepts/urls.md" >}}
[`wire` package]: http://github.com/google/wire

## Sending Messages on a Topic {#sending}

Sending a message on a [Topic](https://godoc.org/github.com/sraphs/gdk/pubsub#Topic) looks
like this:

{{< goexample src="github.com/sraphs/gdk/pubsub.ExampleTopic_Send" imports="0" >}}

Note that the [semantics of message delivery][] can vary by backing service.

[semantics of message delivery]: https://godoc.org/github.com/sraphs/gdk/pubsub#hdr-At_most_once_and_At_least_once_Delivery

## Other Usage Samples

* [CLI Sample](https://github.com/sraphs/gdk/tree/master/samples/gocdk-pubsub)
* [Order Processor sample](https://github.com/sraphs/gdk/tutorials/order/)
* [pubsub package examples](https://godoc.org/github.com/sraphs/gdk/pubsub#pkg-examples)

## Supported Pub/Sub Services {#services}

### RabbitMQ {#rabbitmq}

The GDK can publish to an [AMQP 0.9.1][] fanout exchange, the dialect of
AMQP spoken by [RabbitMQ][]. A RabbitMQ URL only includes the exchange name.
The RabbitMQ's server is discovered from the `RABBIT_SERVER_URL` environment
variable (which is something like `amqp://guest:guest@localhost:5672/`).

{{< goexample "github.com/sraphs/gdk/pubsub/rabbitpubsub.Example_openTopicFromURL" >}}

[AMQP 0.9.1]: https://www.rabbitmq.com/protocol.html
[RabbitMQ]: https://www.rabbitmq.com

#### RabbitMQ Constructor {#rabbitmq-ctor}

The [`rabbitpubsub.OpenTopic`][] constructor opens a RabbitMQ exchange. You
must first create an [`*amqp.Connection`][] to your RabbitMQ instance.

{{< goexample "github.com/sraphs/gdk/pubsub/rabbitpubsub.ExampleOpenTopic" >}}

[`*amqp.Connection`]: https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Connection
[`rabbitpubsub.OpenTopic`]: https://godoc.org/github.com/sraphs/gdk/pubsub/rabbitpubsub#OpenTopic

### NATS {#nats}

The GDK can publish to a [NATS][] subject. A NATS URL only includes the
subject name. The NATS server is discovered from the `NATS_SERVER_URL`
environment variable (which is something like `nats://nats.example.com`).

{{< goexample "github.com/sraphs/gdk/pubsub/natspubsub.Example_openTopicFromURL" >}}

Because NATS does not natively support metadata, messages sent to NATS will
be encoded with [gob][].

[gob]: https://golang.org/pkg/encoding/gob/
[NATS]: https://nats.io/

#### NATS Constructor {#nats-ctor}

The [`natspubsub.OpenTopic`][] constructor opens a NATS subject as a topic. You
must first create an [`*nats.Conn`][] to your NATS instance.

{{< goexample "github.com/sraphs/gdk/pubsub/natspubsub.ExampleOpenTopic" >}}

[`*nats.Conn`]: https://godoc.org/github.com/nats-io/go-nats#Conn
[`natspubsub.OpenTopic`]: https://godoc.org/github.com/sraphs/gdk/pubsub/natspubsub#OpenTopic

### Kafka {#kafka}

The GDK can publish to a [Kafka][] cluster. A Kafka URL only includes the
topic name. The brokers in the Kafka cluster are discovered from the
`KAFKA_BROKERS` environment variable (which is a comma-delimited list of
hosts, something like `1.2.3.4:9092,5.6.7.8:9092`).

{{< goexample "github.com/sraphs/gdk/pubsub/kafkapubsub.Example_openTopicFromURL" >}}

[Kafka]: https://kafka.apache.org/

#### Kafka Constructor {#kafka-ctor}

The [`kafkapubsub.OpenTopic`][] constructor opens a Kafka topic to publish
messages to. Depending on your Kafka cluster configuration (see
`auto.create.topics.enable`), you may need to provision the topic beforehand.

In addition to the list of brokers, you'll need a [`*sarama.Config`][], which
exposes many knobs that can affect performance and semantics; review and set
them carefully. [`kafkapubsub.MinimalConfig`][] provides a minimal config to get
you started.

{{< goexample "github.com/sraphs/gdk/pubsub/kafkapubsub.ExampleOpenTopic" >}}

[`*sarama.Config`]: https://godoc.org/github.com/Shopify/sarama#Config
[`kafkapubsub.OpenTopic`]: https://godoc.org/github.com/sraphs/gdk/pubsub/kafkapubsub#OpenTopic
[`kafkapubsub.MinimalConfig`]: https://godoc.org/github.com/sraphs/gdk/pubsub/kafkapubsub#MinimalConfig

### In-Memory {#mem}

The GDK includes an in-memory Pub/Sub provider useful for local testing.
The names in `mem://` URLs are a process-wide namespace, so subscriptions to
the same name will receive messages posted to that topic. This is detailed
more in the [subscription guide][subscribe-mem].

{{< goexample "github.com/sraphs/gdk/pubsub/mempubsub.Example_openTopicFromURL" >}}

[subscribe-mem]: {{< ref "./subscribe.md#mem" >}}

#### In-Memory Constructor {#mem-ctor}

To create an in-memory Pub/Sub topic, use the [`mempubsub.NewTopic`
function][]. You can use the returned topic to create in-memory
subscriptions, as detailed in the [subscription guide][subscribe-mem-ctor].

{{< goexample "github.com/sraphs/gdk/pubsub/mempubsub.ExampleNewTopic" >}}

[`mempubsub.NewTopic` function]: https://godoc.org/github.com/sraphs/gdk/pubsub/mempubsub#NewTopic
[subscribe-mem-ctor]: {{< ref "./subscribe.md#mem-ctor" >}}


### Redis {#redis}

The GDK can publish to a [Redis][] subject. A Redis URL only includes the
subject name. The Redis server is discovered from the `REDIS_SERVER_URL`
environment variable (which is something like `redis://redis.example.com`).

{{< goexample "github.com/sraphs/gdk/pubsub/redispubsub.Example_openTopicFromURL" >}}

Because Redis does not natively support metadata, messages sent to Redis will
be encoded with [gob][].

[gob]: https://golang.org/pkg/encoding/gob/
[Redis]: https://redis.io/

#### Redis Constructor {#redis-ctor}

The [`redispubsub.OpenTopic`][] constructor opens a Redis subject as a topic. You
must first create an [`*redis.Client`][] to your Redis instance.

{{< goexample "github.com/sraphs/gdk/pubsub/redispubsub.ExampleOpenTopic" >}}

[`*redis.Client`]: https://godoc.org/github.com/go-redis/redis/v8#Client
[`redispubsub.OpenTopic`]: https://godoc.org/github.com/sraphs/gdk/pubsub/redispubsub#OpenTopic


### Pulsar {#pulsar}

The GDK can publish to a [Pulsar][] subject. A Pulsar URL only includes the
subject name. The Pulsar server is discovered from the `Pulsar_SERVER_URL`
environment variable (which is something like `pulsar://localhost:6650`).

{{< goexample "github.com/sraphs/gdk/pubsub/pulsarpubsub.Example_openTopicFromURL" >}}

[Pulsar]: https://pulsar.apache.org/

#### Pulsar Constructor {#pulsar-ctor}

The [`pulsarpubsub.OpenTopic`][] constructor opens a Pulsar topic to publish
messages to.

{{< goexample "github.com/sraphs/gdk/pubsub/pulsarpubsub.ExampleOpenTopic" >}}

[`pulsarpubsub.OpenTopic`]: https://godoc.org/github.com/sraphs/gdk/pubsub/pulsarpubsub#OpenTopic
[`pulsarpubsub.MinimalConfig`]: https://godoc.org/github.com/sraphs/gdk/pubsub/pulsarpubsub#MinimalConfig