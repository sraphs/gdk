package mempubsub

import (
	"context"
	"testing"
	"time"

	"github.com/sraphs/gdk/pubsub/driver"
	"github.com/sraphs/gdk/pubsub/drivertest"
)

type harness struct{}

func newHarness(ctx context.Context, t *testing.T) (drivertest.Harness, error) {
	return &harness{}, nil
}

func (h *harness) CreateTopic(ctx context.Context, testName string) (dt driver.Topic, cleanup func(), err error) {
	cleanup = func() {}
	return &topic{}, cleanup, nil
}

func (h *harness) MakeNonexistentTopic(ctx context.Context) (driver.Topic, error) {
	// A nil *topic behaves like a nonexistent topic.
	return (*topic)(nil), nil
}

func (h *harness) CreateSubscription(ctx context.Context, dt driver.Topic, testName string) (ds driver.Subscription, cleanup func(), err error) {
	ds = newSubscription(dt.(*topic), time.Second)
	cleanup = func() {}
	return ds, cleanup, nil
}

func (h *harness) MakeNonexistentSubscription(ctx context.Context) (driver.Subscription, func(), error) {
	return newSubscription(nil, time.Second), func() {}, nil
}

func (h *harness) Close() {}

func (h *harness) MaxBatchSizes() (int, int) { return 0, 0 }

func (*harness) SupportsMultipleSubscriptions() bool { return true }

func TestConformance(t *testing.T) {
	drivertest.RunConformanceTests(t, newHarness, nil)
}

func BenchmarkMemPubSub(b *testing.B) {
	ctx := context.Background()
	topic := NewTopic()
	defer topic.Shutdown(ctx)
	sub := NewSubscription(topic, time.Second)
	defer sub.Shutdown(ctx)

	drivertest.RunBenchmarks(b, topic, sub)
}
