package kafka

import (
	"context"
	"flag"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/client"
)

var (
	// host 127.0.0.1 test.kafka.com
	ctx    = context.Background()
	target = flag.String("target", "kafka://test.kafka.com:9092?clientid=test_producer",
		"kafka server target dsn like address for producer mode")
)

func TestSendMessage(t *testing.T) {
	flag.Parse()

	proxy := NewClientProxy("trpc.kafka.server.service", client.WithTarget(*target))
	partition, offset, err := proxy.SendMessage(context.Background(), "test_topic", []byte("hello"), []byte("sync"),
		sarama.RecordHeader{
			Key:   []byte("key"),
			Value: []byte("val"),
		})

	t.Logf("SendMessage partition=%v offset=%v err=%v", partition, offset, err)
	assert.Nil(t, nil)
}

func TestAsyncSendMessage(t *testing.T) {
	flag.Parse()

	proxy := NewClientProxy("trpc.kafka.server.service", client.WithTarget(*target))
	err := proxy.AsyncSendMessage(context.Background(), "test_topic", []byte("hello"), []byte("async"), sarama.RecordHeader{})

	t.Logf("AsyncSendMessage err=%v", err)
	assert.Nil(t, nil)
}

func TestProduce(t *testing.T) {
	flag.Parse()

	proxy := NewClientProxy("trpc.kafka.server.service", client.WithTarget(*target))
	err := proxy.Produce(context.Background(), []byte("test_topic"), []byte("hello"), sarama.RecordHeader{
		Key:   []byte("key"),
		Value: []byte("val"),
	})

	t.Logf("Produce err=%v", err)

	assert.Nil(t, nil)
}

func TestKafkaCli_SendSaramaMessage(t *testing.T) {
	flag.Parse()

	proxy := &kafkaCli{Client: client.DefaultClient}
	partition, offset, err := proxy.SendSaramaMessage(ctx, sarama.ProducerMessage{})
	assert.NotNil(t, err)
	assert.Zero(t, partition)
	assert.Zero(t, offset)
}
