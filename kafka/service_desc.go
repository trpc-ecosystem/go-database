package kafka

import (
	"context"

	"github.com/Shopify/sarama"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
)

// KafkaConsumer consumer interface
type KafkaConsumer interface {
	Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}

type kafkaHandler func(ctx context.Context, msg *sarama.ConsumerMessage) error

// Handle main processing function
func (h kafkaHandler) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	return h(ctx, msg)
}

// KafkaConsumerServiceDesc descriptor
var KafkaConsumerServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.kafka.consumer.service",
	HandlerType: ((*KafkaConsumer)(nil)),
	Methods: []server.Method{{
		Name: "/trpc.kafka.consumer.service/handle",
		Func: KafkaConsumerHandle,
	}},
}

// KafkaConsumerHandle consumer service handler wrapper
func KafkaConsumerHandle(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	filters, err := f(nil)
	if err != nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, _ interface{}) (interface{}, error) {
		msg := codec.Message(ctx)
		m, ok := msg.ServerReqHead().(*sarama.ConsumerMessage)
		if !ok {
			return nil, errs.NewFrameError(errs.RetServerDecodeFail, "kafka consumer handler: message type invalid")
		}
		return nil, svr.(KafkaConsumer).Handle(ctx, m)
	}
	return filters.Filter(ctx, nil, handleFunc)
}

// RegisterKafkaConsumerService register service
func RegisterKafkaConsumerService(s server.Service, svr KafkaConsumer) {
	_ = s.Register(&KafkaConsumerServiceDesc, svr)
}

// RegisterKafkaHandlerService register handle
func RegisterKafkaHandlerService(s server.Service,
	handle func(ctx context.Context, msg *sarama.ConsumerMessage) error,
) {
	_ = s.Register(&KafkaConsumerServiceDesc, kafkaHandler(handle))
}

// BatchConsumer batch consumer
type BatchConsumer interface {
	// Handle callback function when a message is received
	Handle(ctx context.Context, msgArray []*sarama.ConsumerMessage) error
}

type batchHandler func(ctx context.Context, msgArray []*sarama.ConsumerMessage) error

// Handle handle
func (h batchHandler) Handle(ctx context.Context, msgArray []*sarama.ConsumerMessage) error {
	return h(ctx, msgArray)
}

// BatchConsumerServiceDesc descriptor for server.RegisterService
var BatchConsumerServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.kafka.consumer.service",
	HandlerType: ((*BatchConsumer)(nil)),
	Methods: []server.Method{
		{
			Name: "/trpc.kafka.consumer.service/handle",
			Func: BatchConsumerHandle,
		},
	},
}

// BatchConsumerHandle batch consumer service handler wrapper
func BatchConsumerHandle(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	filters, err := f(nil)
	if err != nil {
		return nil, err
	}

	handleFunc := func(ctx context.Context, reqbody interface{}) (interface{}, error) {
		msg := codec.Message(ctx)
		msgs, ok := msg.ServerReqHead().([]*sarama.ConsumerMessage)
		if !ok {
			return nil, errs.NewFrameError(errs.RetServerDecodeFail, "kafka consumer handler: message type invalid")
		}
		return nil, svr.(BatchConsumer).Handle(ctx, msgs)
	}

	return filters.Filter(ctx, nil, handleFunc)
}

// RegisterBatchHandlerService register consumer function
func RegisterBatchHandlerService(
	s server.Service,
	handle func(ctx context.Context, msgArray []*sarama.ConsumerMessage) error,
) {
	_ = s.Register(&BatchConsumerServiceDesc, batchHandler(handle))
}
