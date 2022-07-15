package pubsub_test

import (
	"context"
	"testing"
	"time"

	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/driver"
)

type funcTopic struct {
	driver.Topic
	sendBatch func(ctx context.Context, ms []*driver.Message) error
	closed    bool
}

func (t *funcTopic) SendBatch(ctx context.Context, ms []*driver.Message) error {
	return t.sendBatch(ctx, ms)
}

func (t *funcTopic) IsRetryable(error) bool { return false }
func (t *funcTopic) Close() error {
	t.closed = true
	return nil
}

func TestTopicShutdownCanBeCanceledEvenWithHangingSend(t *testing.T) {
	dt := &funcTopic{
		sendBatch: func(ctx context.Context, ms []*driver.Message) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	topic := pubsub.NewTopic(dt, nil)

	go func() {
		m := &pubsub.Message{}
		if err := topic.Send(context.Background(), m); err == nil {
			t.Fatal("nil err from Send, expected context cancellation error")
		}
	}()

	done := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	go func() {
		topic.Shutdown(ctx)
		close(done)
	}()

	// Now cancel the context being used by topic.Shutdown.
	cancel()

	// It shouldn't take too long before topic.Shutdown stops.
	tooLong := 5 * time.Second
	select {
	case <-done:
	case <-time.After(tooLong):
		t.Fatalf("waited too long(%v) for Shutdown(ctx) to run", tooLong)
	}
}

func TestTopicCloseIsCalled(t *testing.T) {
	ctx := context.Background()
	dt := &funcTopic{}
	topic := pubsub.NewTopic(dt, nil)
	topic.Shutdown(ctx)
	if !dt.closed {
		t.Error("want Topic.Close to have been called")
	}
}
