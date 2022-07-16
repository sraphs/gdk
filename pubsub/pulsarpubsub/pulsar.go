package pulsarpubsub

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/sirupsen/logrus"

	"github.com/sraphs/gdk/gdkerr"
	"github.com/sraphs/gdk/pubsub"
	"github.com/sraphs/gdk/pubsub/batcher"
	"github.com/sraphs/gdk/pubsub/driver"
)

var (
	errTopicNotFound  = errors.New("topic not found")
	errNotInitialized = errors.New("pulsarpubsub not initialized")
)

var sendBatcherOpts = &batcher.Options{
	MaxBatchSize: 100,
	MaxHandlers:  2,
}

var recvBatcherOpts = &batcher.Options{
	MaxBatchSize: 1,
	MaxHandlers:  1,
}

// Scheme is the URL scheme pulsarpubsub registers its URLOpeners under on pubsub.DefaultMux.
const Scheme = "pulsar"

func init() {
	o := new(defaultOpener)
	pubsub.DefaultURLMux().RegisterTopic(Scheme, o)
	pubsub.DefaultURLMux().RegisterSubscription(Scheme, o)
}

var _ pubsub.TopicURLOpener = (*defaultOpener)(nil)
var _ pubsub.SubscriptionURLOpener = (*defaultOpener)(nil)

// defaultOpener create a default opener.
type defaultOpener struct {
	init   sync.Once
	opener *URLOpener
	err    error
}

func (o *defaultOpener) defaultOpener(ctx context.Context) (*URLOpener, error) {
	o.init.Do(func() {
		pulsarURL := os.Getenv("PULSAR_SERVER_URL")
		if pulsarURL == "" {
			o.err = errors.New("PULSAR_SERVER_URL environment variable not set")
			return
		}

		config := MinimalConfig(pulsarURL)
		client, err := pulsar.NewClient(config)

		if err != nil {
			o.err = fmt.Errorf("redispubsub: invalid PULSAR_SERVER_URL: %v", err)
			return
		}

		o.opener = &URLOpener{
			Client:              client,
			TopicOptions:        TopicOptions{},
			SubscriptionOptions: SubscriptionOptions{},
		}
	})
	return o.opener, o.err
}

func (o *defaultOpener) OpenTopicURL(ctx context.Context, u *url.URL) (*pubsub.Topic, error) {
	opener, err := o.defaultOpener(ctx)
	if err != nil {
		return nil, fmt.Errorf("open topic %v: failed to open default connection: %v", u, err)
	}
	return opener.OpenTopicURL(ctx, u)
}

func (o *defaultOpener) OpenSubscriptionURL(ctx context.Context, u *url.URL) (*pubsub.Subscription, error) {
	opener, err := o.defaultOpener(ctx)
	if err != nil {
		return nil, fmt.Errorf("open subscription %v: failed to open default connection: %v", u, err)
	}
	return opener.OpenSubscriptionURL(ctx, u)
}

// MinimalConfig returns a minimal pulsar.ClientOptions.
func MinimalConfig(url string) pulsar.ClientOptions {
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.ErrorLevel)

	return pulsar.ClientOptions{
		URL:               url,
		ConnectionTimeout: 3 * time.Second,
		OperationTimeout:  3 * time.Second,
		Logger:            log.NewLoggerWithLogrus(logger),
	}
}

// URLOpener opens Redis URLs like "redis://my-topic".
//
// The URL host+path is used as the subject.
//
// No query parameters are supported.
type URLOpener struct {
	// Client to use for communication with the server.
	Client pulsar.Client

	// TopicOptions specifies the options to pass to OpenTopic.
	TopicOptions TopicOptions
	// SubscriptionOptions specifies the options to pass to OpenSubscription.
	SubscriptionOptions SubscriptionOptions
}

// OpenTopicURL opens a pubsub.Topic based on u.
func (o *URLOpener) OpenTopicURL(ctx context.Context, u *url.URL) (*pubsub.Topic, error) {
	for param := range u.Query() {
		return nil, fmt.Errorf("open topic %v: invalid query parameter %q", u, param)
	}
	topicName := path.Join(u.Host, u.Path)
	o.TopicOptions.Topic = topicName
	return OpenTopic(o.Client, &o.TopicOptions)
}

// OpenSubscriptionURL opens a pubsub.Subscription based on u.
func (o *URLOpener) OpenSubscriptionURL(ctx context.Context, u *url.URL) (*pubsub.Subscription, error) {
	for param, value := range u.Query() {
		switch param {
		case "topic":
			o.SubscriptionOptions.Topics = value
		case "type":
			if len(value) == 0 {
				return nil, fmt.Errorf("open subscription %v: invalid query parameter %q", u, param)
			}
			switch value[0] {
			case "exclusive":
				o.SubscriptionOptions.Type = pulsar.Exclusive
				break
			case "shared":
				o.SubscriptionOptions.Type = pulsar.Shared
				break
			case "failover":
				o.SubscriptionOptions.Type = pulsar.Failover
				break
			case "keyShared":
				o.SubscriptionOptions.Type = pulsar.KeyShared
				break
			default:
				return nil, fmt.Errorf("open subscription %v: invalid subscription type %q", u, value)
			}
		case "position":
			if len(value) == 0 {
				return nil, fmt.Errorf("open subscription %v: invalid query parameter %q", u, param)
			}

			position := value[0]
			switch position {
			case "latest":
				o.SubscriptionOptions.SubscriptionInitialPosition = pulsar.SubscriptionPositionLatest
				break
			case "earliest":
				o.SubscriptionOptions.SubscriptionInitialPosition = pulsar.SubscriptionPositionEarliest
				break
			default:
				return nil, fmt.Errorf("open subscription %v: invalid query parameter %q", u, position)
			}
		default:
			return nil, fmt.Errorf("open subscription %v: invalid query parameter %q", u, param)
		}
	}
	subscriptionName := path.Join(u.Host, u.Path)
	o.SubscriptionOptions.SubscriptionName = subscriptionName
	return OpenSubscription(o.Client, &o.SubscriptionOptions)
}

// TopicOptions sets options for constructing a *pubsub.Topic backed by Pulsar.
type TopicOptions struct {
	pulsar.ProducerOptions
	KeyName string
}

// SubscriptionOptions sets options for constructing a *pubsub.Subscription
// backed by Pulsar.
type SubscriptionOptions struct {
	pulsar.ConsumerOptions
	KeyName string
}

var _ driver.Topic = (*topic)(nil)

type topic struct {
	producer pulsar.Producer
	opts     TopicOptions
}

// OpenTopic returns a *pubsub.Topic for use with Redis.
// The channel is the Redis Chanel; for more info, see
// https://redis.io/commands/pubsub-channels.
func OpenTopic(client pulsar.Client, opts *TopicOptions) (*pubsub.Topic, error) {
	dt, err := openTopic(client, opts)
	if err != nil {
		return nil, err
	}
	return pubsub.NewTopic(dt, sendBatcherOpts), nil
}

// openTopic returns the driver for OpenTopic. This function exists so the test
// harness can get the driver interface implementation if it needs to.
func openTopic(client pulsar.Client, opts *TopicOptions) (driver.Topic, error) {
	if opts == nil {
		opts = &TopicOptions{}
	}

	producer, err := client.CreateProducer(opts.ProducerOptions)
	if err != nil {
		return nil, err
	}
	return &topic{producer: producer, opts: *opts}, nil
}

// SendBatch implements driver.Topic
func (t *topic) SendBatch(ctx context.Context, ms []*driver.Message) error {
	if t == nil || t.producer == nil {
		return errNotInitialized
	}

	if t.opts.Topic == "" {
		return errTopicNotFound
	}

	for _, m := range ms {
		if err := ctx.Err(); err != nil {
			return err
		}

		var key string
		if t.opts.KeyName != "" {
			if k, ok := m.Metadata[t.opts.KeyName]; ok {
				key = k
			}
		}

		pm := &pulsar.ProducerMessage{
			Payload:    m.Body,
			Key:        key,
			Properties: m.Metadata,
			EventTime:  time.Now(),
		}

		if m.BeforeSend != nil {
			asFunc := func(i interface{}) bool {
				if p, ok := i.(**pulsar.ProducerMessage); ok {
					*p = pm
					return true
				}
				return false
			}
			if err := m.BeforeSend(asFunc); err != nil {
				return err
			}
		}

		if _, err := t.producer.Send(ctx, pm); err != nil {
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
	if p, ok := i.(*pulsar.Producer); ok {
		*p = t.producer
		return true
	}
	return false
}

// Close implements driver.Topic
func (t *topic) Close() error {
	if t == nil || t.producer == nil {
		return nil
	}

	t.producer.Close()
	return nil
}

// ErrorAs implements driver.Topic
func (*topic) ErrorAs(err error, i interface{}) bool {
	return errorAs(err, i)
}

// ErrorCode implements driver.Topic
func (*topic) ErrorCode(err error) gdkerr.ErrorCode {
	return errorCode(err)
}

// IsRetryable implements driver.Topic
func (*topic) IsRetryable(err error) bool {
	return false
}

var _ driver.Subscription = (*subscription)(nil)

type subscription struct {
	opts SubscriptionOptions
	sub  pulsar.Consumer
	ch   <-chan pulsar.ConsumerMessage
	mu   sync.Mutex

	receiveBatchHook func() // for testing
}

// OpenSubscription returns a *pubsub.Subscription representing a Redis Subscribe.
// The topicName is the Pulsar Channel to subscribe to;
// for more info, see https://pulsar.apache.org/docs/next/concepts-topic-compaction.
func OpenSubscription(client pulsar.Client, opts *SubscriptionOptions) (*pubsub.Subscription, error) {
	ds, err := openSubscription(client, opts)
	if err != nil {
		return nil, err
	}
	return pubsub.NewSubscription(ds, recvBatcherOpts, nil), nil
}

func openSubscription(client pulsar.Client, opts *SubscriptionOptions) (driver.Subscription, error) {
	if opts == nil {
		opts = &SubscriptionOptions{}
	}

	ch := make(chan pulsar.ConsumerMessage, 100)
	opts.MessageChannel = ch

	pulsarOpts := pulsar.ConsumerOptions(opts.ConsumerOptions)

	sub, err := client.Subscribe(pulsarOpts)

	if err != nil {
		return nil, err
	}

	ps := &subscription{
		opts:             *opts,
		sub:              sub,
		ch:               ch,
		receiveBatchHook: func() {},
	}

	return ps, nil
}

// ReceiveBatch implements driver.Subscription
func (s *subscription) ReceiveBatch(ctx context.Context, maxMessages int) ([]*driver.Message, error) {
	if s == nil || s.sub == nil {
		return nil, errNotInitialized
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.receiveBatchHook()

	// Get up to maxMessages waiting messages, but don't take too long.
	var ms []*driver.Message
	maxTime := time.NewTimer(50 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-maxTime.C:
			return ms, nil
		case msg := <-s.ch:
			ms = append(ms, toMessage(msg))
			if len(ms) >= maxMessages {
				return ms, nil
			}
		}
	}
}

func toMessage(msg pulsar.ConsumerMessage) *driver.Message {
	loggableID := msg.Key()
	if loggableID == "" {
		loggableID = msg.OrderingKey()
	}
	if loggableID == "" {
		loggableID = string(msg.ID().Serialize())
	}
	return &driver.Message{
		LoggableID: loggableID,
		Body:       msg.Payload(),
		AckID:      msg.ID(),
		Metadata:   msg.Properties(),
		AsFunc: func(i interface{}) bool {
			if p, ok := i.(**pulsar.ConsumerMessage); ok {
				*p = &msg
				return true
			}
			return false
		},
	}
}

// As implements driver.Subscription
func (s *subscription) As(i interface{}) bool {
	if p, ok := i.(*pulsar.Consumer); ok {
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

	s.sub.Close()
	return nil
}

// ErrorAs implements driver.Subscription
func (*subscription) ErrorAs(err error, i interface{}) bool {
	return errorAs(err, i)
}

// ErrorCode implements driver.Topic
func (*subscription) ErrorCode(err error) gdkerr.ErrorCode {
	return errorCode(err)
}

// IsRetryable implements driver.Subscription
func (s *subscription) IsRetryable(err error) bool {
	if s == nil {
		return false
	}

	return s.opts.RetryEnable
}

// SendAcks implements driver.Subscription
func (s *subscription) SendAcks(ctx context.Context, ackIDs []driver.AckID) error {
	return s.sendAcksOrNacks(ctx, ackIDs, true)
}

// SendNacks implements driver.Subscription
func (s *subscription) SendNacks(ctx context.Context, ackIDs []driver.AckID) error {
	return s.sendAcksOrNacks(ctx, ackIDs, false)
}

func (s *subscription) sendAcksOrNacks(ctx context.Context, ackIDs []driver.AckID, ack bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ack/Nack calls don't wait for a response, so this loop should execute relatively
	// quickly.
	// It wouldn't help to make it concurrent, because Channel.Ack/Nack grabs a
	// channel-wide mutex. (We could consider using multiple channels if performance
	// becomes an issue.)
	for _, id := range ackIDs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if ack {
			askID, ok := id.(pulsar.MessageID)
			if !ok {
				return fmt.Errorf("invalid AckID type: %T", id)
			}
			s.sub.AckID(askID)
		} else {
			nackID, ok := id.(pulsar.MessageID)
			if !ok {
				return fmt.Errorf("invalid NackID type: %T", id)
			}
			s.sub.NackID(nackID)
		}
	}
	return nil
}

func errorAs(err error, i interface{}) bool {
	switch e := err.(type) {
	case *pulsar.Error:
		if p, ok := i.(**pulsar.Error); ok {
			*p = e
			return true
		}
	}
	return false
}

func errorCode(err error) gdkerr.ErrorCode {
	if err == nil {
		return gdkerr.OK
	}

	switch err {
	case nil:
		return gdkerr.OK
	case context.Canceled:
		return gdkerr.Canceled
	case context.DeadlineExceeded:
		return gdkerr.DeadlineExceeded
	case errNotInitialized, errTopicNotFound:
		return gdkerr.NotFound
	}

	if pe, ok := err.(*pulsar.Error); ok {
		switch pe.Result() {
		case pulsar.Ok:
			return gdkerr.OK
		case pulsar.InvalidConfiguration,
			pulsar.InvalidURL,
			pulsar.InvalidTopicName:
			return gdkerr.InvalidArgument
		case pulsar.TimeoutError:
			return gdkerr.DeadlineExceeded
		case pulsar.LookupError,
			pulsar.TopicNotFound,
			pulsar.SubscriptionNotFound,
			pulsar.ConsumerNotFound:
			return gdkerr.NotFound
		case pulsar.ConnectError,
			pulsar.NotConnectedError,
			pulsar.ConsumerClosed,
			pulsar.ProducerClosed,
			pulsar.AlreadyClosedError:
			return gdkerr.Canceled
		case pulsar.AuthenticationError,
			pulsar.AuthorizationError,
			pulsar.ErrorGettingAuthenticationData:
			return gdkerr.PermissionDenied
		}
	}

	return gdkerr.Unknown
}
