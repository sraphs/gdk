package redispubsub

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-redis/redis/v8"

	"github.com/sraphs/gdk/gdkerr"
	"github.com/sraphs/gdk/internal/testing/setup"
	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/driver"
	"github.com/sraphs/gdk/pubsub/drivertest"
)

const localRedisServerURL = "redis://localhost:6379"

type harness struct {
	rc *redis.Client
}

func newHarness(ctx context.Context, t *testing.T) (drivertest.Harness, error) {
	if !setup.HasDockerTestEnvironment() {
		t.Skip("Skipping Kafka tests since the Kafka server is not available")
	}

	opt, err := redis.ParseURL(localRedisServerURL)
	if err != nil {
		return nil, err
	}

	rc := redis.NewClient(opt)
	return &harness{rc}, nil
}

func (h *harness) CreateTopic(ctx context.Context, testName string) (driver.Topic, func(), error) {
	cleanup := func() {}
	dt, err := openTopic(h.rc, testName)
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
	ds, err := openSubscription(h.rc, testName, nil)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		var sub *redis.PubSub
		if ds.As(&sub) {
			sub.Close()
		}
	}
	return ds, cleanup, nil
}

func (h *harness) MakeNonexistentSubscription(ctx context.Context) (driver.Subscription, func(), error) {
	return (*subscription)(nil), func() {}, nil
}

func (h *harness) Close() {
	// h.rc.Close()
}

func (h *harness) MaxBatchSizes() (int, int) { return 0, 0 }

func (*harness) SupportsMultipleSubscriptions() bool { return true }

type redisAsTest struct{}

func (redisAsTest) Name() string {
	return "redis test"
}

func (redisAsTest) TopicCheck(topic *pubsub.Topic) error {
	var c2 redis.Client
	if topic.As(&c2) {
		return fmt.Errorf("cast succeeded for %T, want failure", &c2)
	}
	var c3 *redis.Client
	if !topic.As(&c3) {
		return fmt.Errorf("cast failed for %T", &c3)
	}
	return nil
}

func (redisAsTest) SubscriptionCheck(sub *pubsub.Subscription) error {
	var c2 redis.PubSub
	if sub.As(&c2) {
		return fmt.Errorf("cast succeeded for %T, want failure", &c2)
	}
	var c3 *redis.PubSub
	if !sub.As(&c3) {
		return fmt.Errorf("cast failed for %T", &c3)
	}
	return nil
}

func (redisAsTest) TopicErrorCheck(t *pubsub.Topic, err error) error {
	var dummy string
	if t.ErrorAs(err, &dummy) {
		return fmt.Errorf("cast succeeded for %T, want failure", &dummy)
	}
	return nil
}

func (redisAsTest) SubscriptionErrorCheck(s *pubsub.Subscription, err error) error {
	var dummy string
	if s.ErrorAs(err, &dummy) {
		return fmt.Errorf("cast succeeded for %T, want failure", &dummy)
	}
	return nil
}

func (redisAsTest) MessageCheck(m *pubsub.Message) error {
	var pm redis.Message
	if m.As(&pm) {
		return fmt.Errorf("cast succeeded for %T, want failure", &pm)
	}
	var ppm *redis.Message
	if !m.As(&ppm) {
		return fmt.Errorf("cast failed for %T", &ppm)
	}
	return nil
}

func (redisAsTest) BeforeSend(as func(interface{}) bool) error {
	return nil
}

func (redisAsTest) AfterSend(as func(interface{}) bool) error {
	return nil
}

func TestConformance(t *testing.T) {
	asTests := []drivertest.AsTest{redisAsTest{}}
	drivertest.RunConformanceTests(t, newHarness, asTests)
}

// These are redispubsub specific to increase coverage.
//
// If we only send a body we should be able to get that from a direct Redis subscriber.
func TestInteropWithDirectRedis(t *testing.T) {
	ctx := context.Background()
	dh, err := newHarness(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	defer dh.Close()
	client := dh.(*harness).rc

	const topic = "foo"
	body := []byte("hello")

	// Send a message using GDK and receive it using Redis directly.
	pt, err := OpenTopic(client, topic, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pt.Shutdown(ctx)

	sub := client.Subscribe(ctx, topic)
	ch := sub.Channel()
	if err = pt.Send(ctx, &pubsub.Message{Body: body}); err != nil {
		t.Fatal(err)
	}
	m := <-ch
	if !bytes.Equal([]byte(m.Payload), body) {
		t.Fatalf("Data did not match. %q vs %q\n", []byte(m.Payload), body)
	}
	sub.Unsubscribe(ctx, topic)

	// Send a message using Redis directly and receive it using GDK.
	ps, err := OpenSubscription(client, topic, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ps.Shutdown(ctx)
	if err := client.Publish(ctx, topic, body).Err(); err != nil {
		t.Fatal(err)
	}
	msg, err := ps.Receive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer msg.Ack()
	if !bytes.Equal(msg.Body, body) {
		t.Fatalf("Data did not match. %q vs %q\n", msg.Body, body)
	}
}

func TestErrorCode(t *testing.T) {
	ctx := context.Background()
	dh, err := newHarness(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	defer dh.Close()
	h := dh.(*harness)

	// Topics
	dt, err := openTopic(h.rc, "bar")
	if err != nil {
		t.Fatal(err)
	}

	if err := dt.ErrorCode(nil); err != gdkerr.OK {
		t.Fatalf("Expected %v, got %v", gdkerr.OK, err)
	}
	if err := dt.ErrorCode(context.Canceled); err != gdkerr.Canceled {
		t.Fatalf("Expected %v, got %v", gdkerr.Canceled, err)
	}
	if err := dt.ErrorCode(redis.Nil); err != gdkerr.NotFound {
		t.Fatalf("Expected %v, got %v", gdkerr.NotFound, err)
	}

	// Subscriptions
	ds, err := openSubscription(h.rc, "bar", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ds.ErrorCode(nil); err != gdkerr.OK {
		t.Fatalf("Expected %v, got %v", gdkerr.OK, err)
	}
	if err := ds.ErrorCode(context.Canceled); err != gdkerr.Canceled {
		t.Fatalf("Expected %v, got %v", gdkerr.Canceled, err)
	}
	if err := ds.ErrorCode(redis.Nil); err != gdkerr.NotFound {
		t.Fatalf("Expected %v, got %v", gdkerr.NotFound, err)
	}
}

func BenchmarkRedisPubSub(b *testing.B) {
	ctx := context.Background()

	opt, err := redis.ParseURL(localRedisServerURL)
	if err != nil {
		b.Fatal(err)
	}

	rc := redis.NewClient(opt)

	defer rc.Close()

	h := &harness{rc}
	dt, cleanup, err := h.CreateTopic(ctx, b.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer cleanup()
	ds, cleanup, err := h.CreateSubscription(ctx, dt, b.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer cleanup()

	topic := pubsub.NewTopic(dt, nil)
	defer topic.Shutdown(ctx)
	sub := pubsub.NewSubscription(ds, nil, nil)
	defer sub.Shutdown(ctx)

	drivertest.RunBenchmarks(b, topic, sub)
}

func fakeConnectionStringInEnv() func() {
	oldEnvVal := os.Getenv("REDIS_SERVER_URL")
	os.Setenv("REDIS_SERVER_URL", localRedisServerURL)
	return func() {
		os.Setenv("REDIS_SERVER_URL", oldEnvVal)
	}
}

func TestOpenTopicFromURL(t *testing.T) {
	ctx := context.Background()
	dh, err := newHarness(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	defer dh.Close()

	cleanup := fakeConnectionStringInEnv()
	defer cleanup()

	tests := []struct {
		URL     string
		WantErr bool
	}{
		// OK.
		{"redis://my-topic", false},
		// Invalid parameter.
		{"redis://my-topic?param=value", true},
	}

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
	ctx := context.Background()
	dh, err := newHarness(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	defer dh.Close()

	cleanup := fakeConnectionStringInEnv()
	defer cleanup()

	tests := []struct {
		URL     string
		WantErr bool
	}{
		// OK.
		{"redis://my-topic", false},
		// Invalid parameter.
		{"redis://my-topic?param=value", true},
	}

	for _, test := range tests {
		sub, err := pubsub.OpenSubscription(ctx, test.URL)
		if (err != nil) != test.WantErr {
			t.Errorf("%s: got error %v, want error %v", test.URL, err, test.WantErr)
		}
		if sub != nil {
			sub.Shutdown(ctx)
		}
	}
}
