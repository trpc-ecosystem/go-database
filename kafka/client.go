package kafka

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
)

// Client kafka interface
type Client interface {
	Produce(ctx context.Context, key, value []byte,
		headers ...sarama.RecordHeader) error
	SendMessage(ctx context.Context, topic string, key, value []byte,
		headers ...sarama.RecordHeader) (partition int32, offset int64, err error)
	AsyncSendMessage(ctx context.Context, topic string, key, value []byte,
		headers ...sarama.RecordHeader) (err error)
	// SendSaramaMessage produce sarama native messages directly
	SendSaramaMessage(ctx context.Context, sMsg sarama.ProducerMessage) (partition int32, offset int64, err error)
}

// kafkaCli backend request struct
type kafkaCli struct {
	ServiceName string
	Client      client.Client
	opts        []client.Option
}

// NewClientProxy create a new kafka backend request proxy.
// The required parameter kafka service name: trpc.kafka.producer.service
var NewClientProxy = func(name string, opts ...client.Option) Client {
	c := &kafkaCli{
		ServiceName: name,
		Client:      client.DefaultClient,
	}

	c.opts = make([]client.Option, 0, len(opts)+2)
	c.opts = append(c.opts, client.WithProtocol("kafka"), client.WithDisableServiceRouter())
	c.opts = append(c.opts, opts...)
	return c
}

// Request kafka request body
type Request struct {
	Topic     string
	Key       []byte
	Value     []byte
	Async     bool // to produce asynchronously or not
	Partition int32
	Headers   []sarama.RecordHeader // Deprecated: use Message.Headers instead
	Message   sarama.ProducerMessage
}

// Response kafka response body
type Response struct {
	Partition int32
	Offset    int64
}

// Produce synchronous production by default, returns whether the sending is successful,
// can be configured async=1 to asynchronous
func (c *kafkaCli) Produce(ctx context.Context, key, value []byte, headers ...sarama.RecordHeader) error {
	req := &Request{
		Message: sarama.ProducerMessage{
			Key:   sarama.ByteEncoder(key),
			Value: sarama.ByteEncoder(value),
		},
	}
	if len(headers) > 0 {
		req.Message.Headers = headers
	}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/produce", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // not serialized
	msg.WithCompressType(0)       // no compression
	msg.WithClientReqHead(req)
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		rsp = &Response{}
		msg.WithClientRspHead(rsp)
		// Generally, users don't care about offset,
		// they only care about whether the sending is successful,
		// and the data that needs offset can be set to return by rsphead
	}

	return c.Client.Invoke(ctx, req, rsp, c.opts...)
}

// SendMessage synchronize production, return the partition and offset value of the data
func (c *kafkaCli) SendMessage(ctx context.Context, topic string, key, value []byte,
	headers ...sarama.RecordHeader,
) (partition int32, offset int64, err error) {
	req := &Request{
		Message: sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.ByteEncoder(key),
			Value: sarama.ByteEncoder(value),
		},
	}
	if len(headers) > 0 {
		req.Message.Headers = headers
	}
	rsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/send", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // not serialized
	msg.WithCompressType(0)       // no compression
	msg.WithClientReqHead(req)
	msg.WithClientRspHead(rsp)

	err = c.Client.Invoke(ctx, req, rsp, c.opts...)
	if err != nil {
		return 0, 0, err
	}

	return rsp.Partition, rsp.Offset, nil
}

// AsyncSendMessage asynchronous production,
// The caller needs to pay attention to the local logs recorded
// after the messages are captured by the success and error channels.
// Consider passing in the callback function later to capture and process this information
func (c *kafkaCli) AsyncSendMessage(ctx context.Context, topic string, key, value []byte,
	headers ...sarama.RecordHeader,
) (err error) {
	req := &Request{
		Async: true,
		Message: sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.ByteEncoder(key),
			Value: sarama.ByteEncoder(value),
		},
	}
	if len(headers) > 0 {
		req.Message.Headers = headers
	}
	rsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/asyncSend", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // not serialized
	msg.WithCompressType(0)       // no compression
	msg.WithClientReqHead(req)
	msg.WithClientRspHead(rsp)

	err = c.Client.Invoke(ctx, req, rsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}

// SendSaramaMessage direct production
func (c *kafkaCli) SendSaramaMessage(
	ctx context.Context,
	sMsg sarama.ProducerMessage,
) (partition int32, offset int64, err error) {
	req := &Request{
		Message: sMsg,
	}
	rsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/sendSarama", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // not serialized
	msg.WithCompressType(0)       // no compression
	msg.WithClientReqHead(req)
	msg.WithClientRspHead(rsp)

	if err := c.Client.Invoke(ctx, req, rsp, c.opts...); err != nil {
		return 0, 0, err
	}

	return rsp.Partition, rsp.Offset, nil
}
