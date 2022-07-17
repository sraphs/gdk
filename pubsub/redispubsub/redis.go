package redispubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/sraphs/gdk/gdkerr"
	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/driver"
)

var redisCtx = context.Background()

// Scheme is the URL scheme redispubsub registers its URLOpeners under on pubsub.DefaultMux.
const Scheme = "redis"

func init() {
	o := new(defaultDialer)
	pubsub.DefaultURLMux().RegisterTopic(Scheme, o)
	pubsub.DefaultURLMux().RegisterSubscription(Scheme, o)
}

var _ pubsub.TopicURLOpener = (*defaultDialer)(nil)
var _ pubsub.SubscriptionURLOpener = (*defaultDialer)(nil)

// defaultDialer dials a default Redis server based on the environment
// variable "REDIS_SERVER_URL".
type defaultDialer struct {
	init   sync.Once
	opener *URLOpener
	err    error
}

func (o *defaultDialer) defaultConn(ctx context.Context) (*URLOpener, error) {
	o.init.Do(func() {
		addr := os.Getenv("REDIS_SERVER_URL")
		if addr == "" {
			o.err = errors.New("REDIS_SERVER_URL environment variable not set")
			return
		}
		opt, err := redis.ParseURL(addr)
		if err != nil {
			o.err = fmt.Errorf("redispubsub: invalid REDIS_SERVER_URL: %v", err)
			return
		}
		client := redis.NewClient(opt)
		o.opener = &URLOpener{Client: client}
	})
	return o.opener, o.err
}

func (o *defaultDialer) OpenTopicURL(ctx context.Context, u *url.URL) (*pubsub.Topic, error) {
	opener, err := o.defaultConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("open topic %v: failed to open default connection: %v", u, err)
	}
	return opener.OpenTopicURL(ctx, u)
}

func (o *defaultDialer) OpenSubscriptionURL(ctx context.Context, u *url.URL) (*pubsub.Subscription, error) {
	opener, err := o.defaultConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("open subscription %v: failed to open default connection: %v", u, err)
	}
	return opener.OpenSubscriptionURL(ctx, u)
}

// URLOpener opens Redis URLs like "redis://my-topic".
//
// The URL host+path is used as the subject.
//
// No query parameters are supported.
type URLOpener struct {
	// Client to use for communication with the server.
	Client *redis.Client
	// TopicOptions specifies the options to pass to OpenTopic.
	TopicOptions TopicOptions
	// SubscriptionOptions specifies the options to pass to OpenSubscription.
	SubscriptionOptions SubscriptionOptions
}

// OpenTopicURL opens a pubsub.Topic based on u.
func (o *URLOpener) OpenTopicURL(ctx context.Context, u *url.URL) (*pubsub.Topic, error) {
	for param := range u.Query() {
		return nil, fmt.Errorf("open topic %v: invalid query parameter %s", u, param)
	}
	channel := path.Join(u.Host, u.Path)
	return OpenTopic(o.Client, channel, &o.TopicOptions)
}

// OpenSubscriptionURL opens a pubsub.Subscription based on u.
func (o *URLOpener) OpenSubscriptionURL(ctx context.Context, u *url.URL) (*pubsub.Subscription, error) {
	opts := o.SubscriptionOptions
	var channels []string
	for param, value := range u.Query() {
		switch param {
		case "topic":
			channels = value
		default:
			return nil, fmt.Errorf("open subscription %v: invalid query parameter %s", u, param)
		}

	}
	nodeID := path.Join(u.Host, u.Path)
	return OpenSubscription(o.Client, nodeID, channels, &opts)
}

// TopicOptions sets options for constructing a *pubsub.Topic backed by Redis.
type TopicOptions struct{}

// SubscriptionOptions sets options for constructing a *pubsub.Subscription
// backed by Redis.
type SubscriptionOptions struct{}

var _ driver.Topic = (*topic)(nil)

type topic struct {
	client  *redis.Client
	channel string
}

// OpenTopic returns a *pubsub.Topic for use with Redis.
// The channel is the Redis Chanel; for more info, see
// https://redis.io/commands/pubsub-channels.
func OpenTopic(rc *redis.Client, channel string, _ *TopicOptions) (*pubsub.Topic, error) {
	dt, err := openTopic(rc, channel)
	if err != nil {
		return nil, err
	}
	return pubsub.NewTopic(dt, nil), nil
}

// openTopic returns the driver for OpenTopic. This function exists so the test
// harness can get the driver interface implementation if it needs to.
func openTopic(rc *redis.Client, channel string) (driver.Topic, error) {
	if rc == nil {
		return nil, errors.New("redispubsub: redis.Client is required")
	}
	return &topic{rc, channel}, nil
}

// SendBatch implements driver.Topic
func (t *topic) SendBatch(ctx context.Context, ms []*driver.Message) error {
	if t == nil || t.client == nil {
		return redis.ErrClosed
	}

	for _, m := range ms {
		if err := ctx.Err(); err != nil {
			return err
		}
		payload, err := encodeMessage(m)
		if err != nil {
			return err
		}
		if m.BeforeSend != nil {
			asFunc := func(i interface{}) bool { return false }
			if err := m.BeforeSend(asFunc); err != nil {
				return err
			}
		}
		if err := t.client.Publish(ctx, t.channel, payload).Err(); err != nil {
			return err
		}
		if m.AfterSend != nil {
			asFunc := func(i interface{}) bool { return false }
			if err := m.AfterSend(asFunc); err != nil {
				return err
			}
		}
	}

	return nil
}

// As implements driver.Topic
func (t *topic) As(i interface{}) bool {
	c, ok := i.(**redis.Client)
	if !ok {
		return false
	}
	*c = t.client
	return true
}

// Close implements driver.Topic
func (t *topic) Close() error {
	return nil
}

// ErrorAs implements driver.Topic
func (*topic) ErrorAs(error, interface{}) bool {
	return false
}

// ErrorCode implements driver.Topic
func (*topic) ErrorCode(err error) gdkerr.ErrorCode {
	switch err {
	case nil:
		return gdkerr.OK
	case context.Canceled:
		return gdkerr.Canceled
	case redis.Nil, redis.ErrClosed:
		return gdkerr.NotFound
	}
	return gdkerr.Unknown
}

// IsRetryable implements driver.Topic
func (*topic) IsRetryable(err error) bool {
	return false
}

var _ driver.Subscription = (*subscription)(nil)

type subscription struct {
	nodeID   string
	channels []string
	sub      *redis.PubSub
	ch       <-chan *redis.Message
	done     chan struct{}
	nextID   int
}

// OpenSubscription returns a *pubsub.Subscription representing a Redis Subscribe.
// The channel is the Redis Channel to subscribe to;
// for more info, see https://redis.io/commands/pubsub-channels/.
func OpenSubscription(rc *redis.Client, nodeID string, channels []string, opts *SubscriptionOptions) (*pubsub.Subscription, error) {
	ds, err := openSubscription(rc, nodeID, channels, opts)
	if err != nil {
		return nil, err
	}
	return pubsub.NewSubscription(ds, nil, nil), nil
}

func openSubscription(rc *redis.Client, nodeID string, channels []string, opts *SubscriptionOptions) (driver.Subscription, error) {
	sub := rc.Subscribe(redisCtx, channels...)

	ps := &subscription{
		nodeID:   nodeID,
		channels: channels,
		sub:      sub,
		ch:       sub.Channel(),
		done:     make(chan struct{}, 1),
		nextID:   1,
	}

	// ensure that channel is initial synced
	time.Sleep(100 * time.Millisecond)

	return ps, nil
}

// ReceiveBatch implements driver.Subscription
func (s *subscription) ReceiveBatch(ctx context.Context, maxMessages int) ([]*driver.Message, error) {
	if s == nil || s.sub == nil {
		return nil, redis.ErrClosed
	}

	// Get up to maxMessages waiting messages, but don't take too long.
	var ms []*driver.Message
	maxTime := time.NewTimer(50 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.done:
			return nil, redis.ErrClosed
		case <-maxTime.C:
			return ms, nil
		case msg := <-s.ch:
			dm, err := decode(msg)
			if err != nil {
				return nil, err
			}
			dm.LoggableID = fmt.Sprintf("msg #%d", s.nextID)
			s.nextID++
			ms = append(ms, dm)
			if len(ms) >= maxMessages {
				return ms, nil
			}
		}
	}
}

// Convert Redis msgs to *driver.Message.
func decode(msg *redis.Message) (*driver.Message, error) {
	if msg == nil {
		return nil, redis.Nil
	}
	var dm driver.Message
	if err := decodeMessage([]byte(msg.Payload), &dm); err != nil {
		return nil, err
	}
	dm.AckID = -1 // Not applicable to Redis
	dm.AsFunc = messageAsFunc(msg)
	return &dm, nil
}

func messageAsFunc(msg *redis.Message) func(interface{}) bool {
	return func(i interface{}) bool {
		p, ok := i.(**redis.Message)
		if !ok {
			return false
		}
		*p = msg
		return true
	}
}

// As implements driver.Subscription
func (s *subscription) As(i interface{}) bool {
	if p, ok := i.(**redis.PubSub); ok {
		*p = s.sub
		return true
	}
	return false
}

// CanNack implements driver.Subscription
func (*subscription) CanNack() bool {
	return false
}

// Close implements driver.Subscription
func (s *subscription) Close() error {
	if s == nil || s.sub == nil {
		return nil
	}

	s.sub.Unsubscribe(redisCtx, s.channels...)
	s.done <- struct{}{}
	return nil
}

// ErrorAs implements driver.Subscription
func (*subscription) ErrorAs(error, interface{}) bool {
	return false
}

// ErrorCode implements driver.Topic
func (*subscription) ErrorCode(err error) gdkerr.ErrorCode {
	switch err {
	case nil:
		return gdkerr.OK
	case context.Canceled:
		return gdkerr.Canceled
	case redis.Nil, redis.ErrClosed:
		return gdkerr.NotFound
	}
	return gdkerr.Unknown
}

// IsRetryable implements driver.Subscription
func (*subscription) IsRetryable(err error) bool {
	return false
}

// SendAcks implements driver.Subscription
func (*subscription) SendAcks(ctx context.Context, ackIDs []driver.AckID) error {
	return nil
}

// SendNacks implements driver.Subscription
func (*subscription) SendNacks(ctx context.Context, ackIDs []driver.AckID) error {
	return nil
}

func encodeMessage(dm *driver.Message) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if len(dm.Metadata) == 0 {
		return dm.Body, nil
	}
	if err := enc.Encode(dm.Metadata); err != nil {
		return nil, err
	}
	if err := enc.Encode(dm.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeMessage(data []byte, dm *driver.Message) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&dm.Metadata); err != nil {
		// This may indicate a normal NATS message, so just treat as the body.
		dm.Metadata = nil
		dm.Body = data
		return nil
	}
	return dec.Decode(&dm.Body)
}
