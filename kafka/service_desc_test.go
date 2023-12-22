package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/filter"
)

type testServer struct{}

func (ts testServer) Register(interface{}, interface{}) error {
	return nil
}

func (ts testServer) Serve() error {
	return nil
}

func (ts testServer) Close(chan struct{}) error {
	return nil
}

// errHandle KafkaConsumerHandle return error
var errHandle = func(interface{}) (filter.ServerChain, error) {
	return filter.ServerChain{filter.NoopServerFilter}, errors.New("fake err")
}

// successHandle KafkaConsumerHandle return nil
var successHandle = func(interface{}) (filter.ServerChain, error) {
	return filter.ServerChain{filter.NoopServerFilter}, nil
}

func TestKafkaConsumerHandle(t *testing.T) {

	t.Run("case header wrong", func(t *testing.T) {
		var kh kafkaHandler = func(context.Context, *sarama.ConsumerMessage) error {
			return nil
		}
		ctx := trpc.BackgroundContext()
		_, err := KafkaConsumerHandle(kh, ctx, successHandle)
		assert.NotNil(t, err)
		ctx = trpc.BackgroundContext()
		_, err = KafkaConsumerHandle(kh, ctx, errHandle)
		assert.NotNil(t, err)
	})

	t.Run("case handle failed", func(t *testing.T) {
		var kh kafkaHandler = func(context.Context, *sarama.ConsumerMessage) error {
			return errors.New("fake error")
		}
		ctx := trpc.BackgroundContext()
		msg := trpc.Message(ctx)
		kafkaMsg := new(sarama.ConsumerMessage)
		msg.WithServerReqHead(kafkaMsg)
		_, err := KafkaConsumerHandle(kh, ctx, successHandle)
		assert.NotNil(t, err)
	})

	t.Run("case success", func(t *testing.T) {
		var kh kafkaHandler = func(ctx context.Context, msg *sarama.ConsumerMessage) error {
			return nil
		}
		ctx := trpc.BackgroundContext()
		msg := trpc.Message(ctx)
		kafkaMsg := new(sarama.ConsumerMessage)
		msg.WithServerReqHead(kafkaMsg)
		_, err := KafkaConsumerHandle(kh, ctx, successHandle)
		assert.Nil(t, err)
	})
}

func TestRegisterKafkaHandlerService(t *testing.T) {
	var kh kafkaHandler = func(context.Context, *sarama.ConsumerMessage) error {
		return nil
	}
	RegisterKafkaConsumerService(testServer{}, kh)
	err := kh.Handle(trpc.BackgroundContext(), new(sarama.ConsumerMessage))
	assert.Nil(t, err)
}

func TestRegisterKafkaConsumerService(t *testing.T) {
	var kh kafkaHandler = func(context.Context, *sarama.ConsumerMessage) error {
		return nil
	}
	RegisterKafkaConsumerService(testServer{}, kh)
}

func TestBatchConsumerHandle(t *testing.T) {
	t.Run("case header wrong", func(t *testing.T) {
		var bh batchHandler = func(context.Context, []*sarama.ConsumerMessage) error {
			return nil
		}
		ctx := trpc.BackgroundContext()
		_, err := BatchConsumerHandle(bh, ctx, successHandle)
		assert.NotNil(t, err)
		ctx = trpc.BackgroundContext()
		_, err = BatchConsumerHandle(bh, ctx, errHandle)
		assert.NotNil(t, err)
	})

	t.Run("case handle failed", func(t *testing.T) {
		var bh batchHandler = func(context.Context, []*sarama.ConsumerMessage) error {
			return errors.New("fake error")
		}
		ctx := trpc.BackgroundContext()
		msg := trpc.Message(ctx)
		kafkaMsg := make([]*sarama.ConsumerMessage, 1)
		msg.WithServerReqHead(kafkaMsg)
		_, err := BatchConsumerHandle(bh, ctx, successHandle)
		assert.NotNil(t, err)
	})

	t.Run("case success", func(t *testing.T) {
		var bh batchHandler = func(context.Context, []*sarama.ConsumerMessage) error {
			return nil
		}
		ctx := trpc.BackgroundContext()
		msg := trpc.Message(ctx)
		kafkaMsg := make([]*sarama.ConsumerMessage, 1)
		msg.WithServerReqHead(kafkaMsg)
		_, err := BatchConsumerHandle(bh, ctx, successHandle)
		assert.Nil(t, err)
	})
}

func TestRegisterBatchHandlerService(t *testing.T) {
	var bh batchHandler = func(context.Context, []*sarama.ConsumerMessage) error {
		return nil
	}
	RegisterBatchHandlerService(testServer{}, bh)
	err := bh.Handle(trpc.BackgroundContext(), []*sarama.ConsumerMessage{})
	assert.Nil(t, err)
}
