package pulsarpubsub

// To run these tests against a real Pulsar server, run localpulsar.sh.
// See https://pulsar.apache.org/docs/next/getting-started-docker for more on the docker container
// that the script runs.

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/sraphs/gdk/internal/testing/setup"
	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/driver"
	"github.com/sraphs/gdk/pubsub/drivertest"
)

var (
	localPulsarURL = "pulsar://localhost:6650"
)

type harness struct {
	client    pulsar.Client
	uniqueID  int
	numSubs   uint32
	numTopics uint32
}

func newHarness(ctx context.Context, t *testing.T) (drivertest.Harness, error) {
	if !setup.HasDockerTestEnvironment() {
		t.Skip("Skipping tests since the server is not available")
	}

	// Create the topic.
	config := MinimalConfig(localPulsarURL)
	client, err := pulsar.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &harness{client: client, uniqueID: rand.Int()}, nil
}

func (h *harness) CreateTopic(ctx context.Context, testName string) (driver.Topic, func(), error) {
	cleanup := func() {}
	topicName := fmt.Sprintf("%s-topic-%d-%d", sanitize(testName), h.uniqueID, atomic.AddUint32(&h.numTopics, 1))
	dt, err := openTopic(h.client, &TopicOptions{
		ProducerOptions: pulsar.ProducerOptions{
			Topic: topicName,
		},
		KeyName: "",
	})
	if err != nil {
		return nil, nil, err
	}
	return dt, cleanup, nil
}

func (h *harness) MakeNonexistentTopic(ctx context.Context) (driver.Topic, error) {
	// A nil *topic behaves like a nonexistent topic.
	return (*topic)(nil), nil
}

func (h *harness) CreateSubscription(ctx context.Context, dt driver.Topic, testName string) (driver.Subscription, func(), error) {
	subscriptionName := fmt.Sprintf("%s-sub-%d-%d", sanitize(testName), h.uniqueID, atomic.AddUint32(&h.numSubs, 1))
	ds, err := openSubscription(h.client, &SubscriptionOptions{
		ConsumerOptions: pulsar.ConsumerOptions{
			SubscriptionName: subscriptionName,
			Topics:           []string{dt.(*topic).opts.Topic},
		},
		KeyName: "",
	})
	return ds, func() {}, err
}

func (h *harness) MakeNonexistentSubscription(ctx context.Context) (driver.Subscription, func(), error) {
	return (*subscription)(nil), func() {}, nil
}

func (h *harness) Close() {
	h.client.Close()
}

func (h *harness) MaxBatchSizes() (int, int) { return sendBatcherOpts.MaxBatchSize, 0 }

func (*harness) SupportsMultipleSubscriptions() bool { return true }

type asTest struct{}

func (asTest) Name() string {
	return "pulsar"
}

func (asTest) TopicCheck(topic *pubsub.Topic) error {
	var sp pulsar.Producer
	if !topic.As(&sp) {
		return fmt.Errorf("cast failed for %T", sp)
	}
	return nil
}

func (asTest) SubscriptionCheck(sub *pubsub.Subscription) error {
	var cg pulsar.Consumer
	if !sub.As(&cg) {
		return fmt.Errorf("cast failed for %T", cg)
	}
	return nil
}

func (asTest) TopicErrorCheck(t *pubsub.Topic, err error) error {
	var dummy string
	if t.ErrorAs(err, &dummy) {
		return fmt.Errorf("cast succeeded for %T, want failure", &dummy)
	}
	return nil
}

func (asTest) SubscriptionErrorCheck(s *pubsub.Subscription, err error) error {
	var dummy string
	if s.ErrorAs(err, &dummy) {
		return fmt.Errorf("cast succeeded for %T, want failure", &dummy)
	}
	return nil
}

func (asTest) MessageCheck(m *pubsub.Message) error {
	var cm *pulsar.ConsumerMessage
	if !m.As(&cm) {
		return fmt.Errorf("cast failed for %T", cm)
	}
	return nil
}

func (asTest) BeforeSend(as func(interface{}) bool) error {
	var pm *pulsar.ProducerMessage
	if !as(&pm) {
		return fmt.Errorf("cast failed for %T", &pm)
	}
	return nil
}

func (asTest) AfterSend(as func(interface{}) bool) error {
	return nil
}

func TestConformance(t *testing.T) {
	asTests := []drivertest.AsTest{asTest{}}
	drivertest.RunConformanceTests(t, newHarness, asTests)
}

// TestKey tests sending/receiving a message with the message key set.
func TestKey(t *testing.T) {
	if !setup.HasDockerTestEnvironment() {
		t.Skip("Skipping tests since the server is not available")
	}
	const (
		keyName  = "pulsarkey"
		keyValue = "pulsarkeyvalue"
	)
	uniqueID := rand.Int()
	ctx := context.Background()

	topicName := fmt.Sprintf("%s-topic-%d", sanitize(t.Name()), uniqueID)
	// Create the topic.
	config := MinimalConfig(localPulsarURL)
	client, err := pulsar.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	topic, err := OpenTopic(client, &TopicOptions{
		ProducerOptions: pulsar.ProducerOptions{
			Topic: topicName,
		},
		KeyName: keyName,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := topic.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	}()

	subscriptionName := fmt.Sprintf("%s-sub-%d", sanitize(t.Name()), uniqueID)
	sub, err := OpenSubscription(client, &SubscriptionOptions{
		ConsumerOptions: pulsar.ConsumerOptions{
			SubscriptionName: subscriptionName,
			Topics:           []string{topicName},
		},
		KeyName: keyName,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := sub.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	}()

	m := &pubsub.Message{
		Metadata: map[string]string{
			"foo":   "bar",
			keyName: keyValue,
		},
		Body: []byte("hello world"),
		BeforeSend: func(as func(interface{}) bool) error {
			// Verify that the Key field was set correctly on the outgoing message.
			var pm *pulsar.ProducerMessage
			if !as(&pm) {
				return errors.New("failed to convert to ProducerMessage")
			}
			gotKey := pm.Key
			if gotKey := string(gotKey); gotKey != keyValue {
				return errors.New("Pulsar key wasn't set appropriately")
			}
			return nil
		},
	}
	err = topic.Send(ctx, m)
	if err != nil {
		t.Fatal(err)
	}

	// The test will hang here if the message isn't available, so use a shorter timeout.
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	got, err := sub.Receive(ctx2)
	if err != nil {
		t.Fatal(err)
	}
	got.Ack()

	m.BeforeSend = nil // don't expect this in the received message
	m.LoggableID = keyValue
	if diff := cmp.Diff(got, m, cmpopts.IgnoreUnexported(pubsub.Message{})); diff != "" {
		t.Errorf("got\n%v\nwant\n%v\ndiff\n%v", got, m, diff)
	}

	// Verify that Key was set in the received message via As.
	var cm *pulsar.ConsumerMessage
	if !got.As(&cm) {
		t.Fatal("failed to get message As ConsumerMessage")
	}
	if gotKey := cm.Key(); gotKey != keyValue {
		t.Errorf("got key %q want %q", gotKey, keyValue)
	}
}

// TestShared tests use of a topic with multiple partitions, including the
// rebalancing that happens when a new consumer appears in the group.
func TestShared(t *testing.T) {
	if !setup.HasDockerTestEnvironment() {
		t.Skip("Skipping tests since the server is not available")
	}
	const (
		keyName   = "pulsarkey"
		nMessages = 10
	)
	uniqueID := rand.Int()
	ctx := context.Background()

	// Create a topic with 10 partitions. Using 10 instead of just 2 because
	// that also tests having multiple claims.
	topicName := fmt.Sprintf("%s-topic-%d", sanitize(t.Name()), uniqueID)
	// Create the topic.
	config := MinimalConfig(localPulsarURL)
	client, err := pulsar.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	topic, err := OpenTopic(client, &TopicOptions{
		ProducerOptions: pulsar.ProducerOptions{
			Topic: topicName,
		},
		KeyName: keyName,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := topic.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	}()

	// Open a subscription.
	subscriptionName := fmt.Sprintf("%s-sub-%d", sanitize(t.Name()), uniqueID)

	sub, err := OpenSubscription(client, &SubscriptionOptions{
		ConsumerOptions: pulsar.ConsumerOptions{
			SubscriptionName: subscriptionName,
			Topic:            topicName,
			Type:             pulsar.Shared,
		},
		KeyName: keyName,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := sub.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	}()

	// Send some messages.
	send := func() {
		for i := 0; i < nMessages; i++ {
			m := &pubsub.Message{
				Metadata: map[string]string{
					keyName: fmt.Sprintf("key%d", i),
				},
				Body: []byte("hello world"),
			}
			if err := topic.Send(ctx, m); err != nil {
				t.Fatal(err)
			}
		}
	}
	send()

	// Receive the messages via the subscription.
	got := make(chan int)
	done := make(chan error)
	read := func(ctx context.Context, subNum int, sub *pubsub.Subscription) {
		for {
			m, err := sub.Receive(ctx)
			if err != nil {
				if err == context.Canceled {
					// Expected after all messages are received, no error.
					done <- nil
				} else {
					done <- err
				}
				return
			}
			m.Ack()
			got <- subNum
		}
	}
	// The test will hang here if the messages aren't available, so use a shorter
	// timeout.
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	go read(ctx2, 0, sub)
	for i := 0; i < nMessages; i++ {
		select {
		case <-got:
		case err := <-done:
			// Premature error.
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	// Add another subscription to the same group. Pulsar will rebalance the
	// consumer group, causing the Cleanup/Setup/ConsumeClaim loop. Each of the
	// two subscriptions should get claims for 50% of the partitions.
	sub2, err := OpenSubscription(client, &SubscriptionOptions{
		ConsumerOptions: pulsar.ConsumerOptions{
			SubscriptionName: subscriptionName,
			Topic:            topicName,
			Type:             pulsar.Shared,
		},
		KeyName: keyName,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := sub2.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	}()

	// Send and receive some messages.
	// Now both subscriptions should get some messages.
	send()

	// The test will hang here if the message isn't available, so use a shorter timeout.
	ctx3, cancel := context.WithTimeout(ctx, 30*time.Second)
	go read(ctx3, 0, sub)
	go read(ctx3, 1, sub2)
	counts := []int{0, 0}
	for i := 0; i < nMessages; i++ {
		select {
		case sub := <-got:
			counts[sub]++
		case err := <-done:
			// Premature error.
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	cancel()
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if counts[0] == 0 || counts[1] == 0 {
		t.Errorf("one of the partitioned subscriptions didn't get any messages: %v", counts)
	}
}

func sanitize(testName string) string {
	return strings.Replace(testName, "/", "_", -1)
}

func BenchmarkPulsar(b *testing.B) {
	ctx := context.Background()
	uniqueID := rand.Int()

	// Create the topic.
	topicName := fmt.Sprintf("%s-topic-%d", b.Name(), uniqueID)
	// Create the topic.
	config := MinimalConfig(localPulsarURL)
	client, err := pulsar.NewClient(config)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	topic, err := OpenTopic(client, &TopicOptions{
		ProducerOptions: pulsar.ProducerOptions{
			Topic: topicName,
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := topic.Shutdown(ctx); err != nil {
			b.Error(err)
		}
	}()

	subscriptionName := fmt.Sprintf("%s-subscription-%d", b.Name(), uniqueID)
	sub, err := OpenSubscription(client, &SubscriptionOptions{
		ConsumerOptions: pulsar.ConsumerOptions{
			SubscriptionName: subscriptionName,
			Topic:            topicName,
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	defer sub.Shutdown(ctx)

	drivertest.RunBenchmarks(b, topic, sub)
}

func fakeConnectionStringInEnv() func() {
	oldEnvVal := os.Getenv("PULSAR_SERVER_URL")
	os.Setenv("PULSAR_SERVER_URL", localPulsarURL)
	return func() {
		os.Setenv("PULSAR_SERVER_URL", oldEnvVal)
	}
}

func TestOpenTopicFromURL(t *testing.T) {
	cleanup := fakeConnectionStringInEnv()
	defer cleanup()

	tests := []struct {
		URL     string
		WantErr bool
	}{
		// OK.
		{"pulsar://my-topic", false},
		// Invalid parameter.
		{"pulsar://my-topic?param=value", true},
	}

	ctx := context.Background()
	for _, test := range tests {
		topic, err := pubsub.OpenTopic(ctx, test.URL)
		if (err != nil) != test.WantErr {
			t.Errorf("%s: got error %v, want error %v", test.URL, err, test.WantErr)
		}
		if topic != nil {
			topic.Shutdown(ctx)
		}
	}
}

func TestOpenSubscriptionFromURL(t *testing.T) {
	cleanup := fakeConnectionStringInEnv()
	defer cleanup()

	tests := []struct {
		URL     string
		WantErr bool
	}{
		// OK.
		{"pulsar://my-sub?topic=my-topic", false},
		// OK, specifying initial position.
		{"pulsar://my-sub?topic=my-topic&position=latest", false},
		{"pulsar://my-sub?topic=my-topic&position=earliest", false},
		// Invalid position specified
		{"pulsar://my-sub?topic=my-topic&position=value", true},
		// Invalid parameter.
		{"pulsar://my-sub?topic=my-topic&param=value", true},
	}

	ctx := context.Background()
	const ignore = "pulsar: client has run out of available brokers to talk to"

	for _, test := range tests {
		sub, err := pubsub.OpenSubscription(ctx, test.URL)
		if err != nil && strings.HasPrefix(err.Error(), ignore) {
			// Since we don't have a real pulsar broker to talk to, we will always get an error when
			// opening a subscription. This test is checking specifically for query parameter usage, so
			// we treat the "no brokers" error message as a nil error.
			err = nil
		}

		if (err != nil) != test.WantErr {
			t.Errorf("%s: got error %v, want error %v", test.URL, err, test.WantErr)
		}
		if sub != nil {
			sub.Shutdown(ctx)
		}
	}
}
