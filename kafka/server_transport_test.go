package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestNewServerTransport(t *testing.T) {
	_ = NewServerTransport(transport.WithReusePort(false))
	assert.Nil(t, nil)
}

func fakeNewConsumerGroup(addrs []string, groupID string, config *sarama.Config) (sarama.ConsumerGroup, error) {
	return testGroup{}, nil
}

func TestListenAndServe(t *testing.T) {
	newConsumerGroup = fakeNewConsumerGroup
	defer func() {
		newConsumerGroup = sarama.NewConsumerGroup
	}()

	ctx2, cancel := context.WithCancel(context.Background())
	address := "test1.kafka.com:1234,test2.kafka.com:5678" +
		"?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest"
	_ = NewServerTransport().ListenAndServe(ctx2, transport.WithListenAddress(address))
	time.Sleep(time.Second)
	cancel()

	ctx2, cancel = context.WithCancel(context.Background())
	address = "test1.kafka.com:1234,test2.kafka.com:5678" +
		"?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&batch=10"
	_ = NewServerTransport().ListenAndServe(ctx, transport.WithListenAddress(address))
	address = "test1.kafka.com:1234,test2.kafka.com:5678" +
		"?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&mechanism=PLAIN&user=ckafka_id#username" +
		"&password=1234&maxMessageBytes=35326236&initial=oldest"
	err := NewServerTransport().ListenAndServe(ctx2, transport.WithListenAddress(address))
	assert.Nil(t, err)

	address = "test1.kafka.com:1234,test2.kafka.com:5678?limiterRate=2&batch=3&batchFlush=1000"
	err = NewServerTransport().ListenAndServe(ctx2, transport.WithListenAddress(address))
	assert.Nil(t, err)

	address = "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678" +
		"&fetchMax=54455&mechanism=xxxxxx&user=ckafka_id#username" +
		"&password=1234&maxMessageBytes=35326236&initial=oldest"
	err = NewServerTransport().ListenAndServe(ctx2, transport.WithListenAddress(address))
	assert.EqualError(t, err, "type:framework, code:131, msg:kafka scram_client.config failed,x.mechanism unknown(xxxxxx)")
	time.Sleep(time.Second)
	assert.Nil(t, nil)
	cancel()
}
