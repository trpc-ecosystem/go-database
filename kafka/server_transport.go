package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"golang.org/x/time/rate"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

var newConsumerGroup = sarama.NewConsumerGroup

func init() {
	transport.RegisterServerTransport("kafka", DefaultServerTransport)
}

// DefaultServerTransport ServerTransport default implement
var DefaultServerTransport = NewServerTransport()

// NewServerTransport build serverTransport
func NewServerTransport(opt ...transport.ServerTransportOption) transport.ServerTransport {
	// option default value
	kafkaOpts := &transport.ServerTransportOptions{}
	for _, o := range opt {
		o(kafkaOpts)
	}
	return &ServerTransport{opts: kafkaOpts}
}

// ServerTransport kafka consumer transport
type ServerTransport struct {
	opts *transport.ServerTransportOptions
}

// ListenAndServe start the listener, and return an error if the listener fails
func (s *ServerTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) (err error) {
	lsOpts := &transport.ListenServeOptions{}
	for _, opt := range opts {
		opt(lsOpts)
	}
	// parse address option
	kafkaUserConfig, err := ParseAddress(lsOpts.Address)
	if err != nil {
		return err
	}

	// get sarama config infomation
	config := kafkaUserConfig.getServerConfig()
	err = kafkaUserConfig.ScramClient.config(config)
	if err != nil {
		return err
	}
	// create custom group connection broker, failure will return an error
	consumerGroup, err := newConsumerGroup(kafkaUserConfig.Brokers, kafkaUserConfig.Group, config)
	if err != nil {
		return err
	}
	limiter := newLimiter(kafkaUserConfig.RateLimitConfig)
	// create consumer
	var handler sarama.ConsumerGroupHandler
	if kafkaUserConfig.BatchConsumeCount > 0 {
		handler = &batchConsumerHandler{
			opts:          lsOpts,
			ctx:           ctx,
			maxNum:        kafkaUserConfig.BatchConsumeCount,
			flushInterval: kafkaUserConfig.BatchFlush,
			retryMax:      kafkaUserConfig.MaxRetry,
			retryInterval: kafkaUserConfig.RetryInterval,
			trpcMeta:      kafkaUserConfig.TrpcMeta,
			limiter:       limiter,
		}
	} else {
		handler = &singleConsumerHandler{
			opts:          lsOpts,
			ctx:           ctx,
			retryMax:      kafkaUserConfig.MaxRetry,
			retryInterval: kafkaUserConfig.RetryInterval,
			trpcMeta:      kafkaUserConfig.TrpcMeta,
			limiter:       limiter,
		}
	}

	go func() {
		// after the service ends, close the group and client
		defer func() {
			if err := consumerGroup.Close(); err != nil {
				log.Errorf("kafka consumerGroup close return err: %s", err)
			}
		}()

		for {
			// only when the handler cleanup will report an error, see sarama.consumerGroup.Consume
			if err := consumerGroup.Consume(ctx, kafkaUserConfig.Topics, handler); err != nil {
				log.ErrorContextf(ctx, "kafka server transport: Consume get error:%v", err)
			}

			select {
			case <-ctx.Done(): // the service has ended, exit the service
				log.ErrorContextf(ctx, "kafka server transport: context done:%v, close", ctx.Err())
				return
			default:
			}
		}
	}()
	return nil
}

// newLimiter get a *rate.Limiter
func newLimiter(conf *RateLimitConfig) *rate.Limiter {
	if conf == nil {
		return rate.NewLimiter(rate.Inf, 0)
	}
	return rate.NewLimiter(rate.Limit(conf.Rate), conf.Burst)
}
