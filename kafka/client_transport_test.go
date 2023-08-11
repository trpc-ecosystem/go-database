// config_parser config information to Userconfig
package kafka

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/transport"
)

func fakeNewSyncProducer(addrs []string, config *sarama.Config) (sarama.SyncProducer, error) {
	return testSyncProducer{}, nil
}

func fakeNewAsyncProducer(addrs []string, conf *sarama.Config) (sarama.AsyncProducer, error) {
	return testAsyncProducer{}, nil
}

func fakeNewBlockedAsyncProducer(addrs []string, conf *sarama.Config) (sarama.AsyncProducer, error) {
	return testBlockedAsyncProducer{}, nil
}

func TestNewClientTransport(t *testing.T) {
	_ = NewClientTransport(transport.WithClientUDPRecvSize(65536))
	assert.Nil(t, nil)
}

func TestRoundTrip(t *testing.T) {
	newSyncProducer = fakeNewSyncProducer
	newAsyncProducer = fakeNewAsyncProducer
	defer func() {
		newSyncProducer = sarama.NewSyncProducer
		newAsyncProducer = sarama.NewAsyncProducer
	}()

	address := "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&trpcMeta=true"
	// async=false
	ctx, cancel := context.WithDeadline(trpc.BackgroundContext(), time.Now().Add(time.Second))
	req := &Request{
		Message: sarama.ProducerMessage{
			Key:   sarama.ByteEncoder([]byte("key")),
			Value: sarama.ByteEncoder([]byte("value")),
		},
	}
	trpc.SetMetaData(ctx, string("key"), []byte("value"))

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/produce", "testServiceName"))
	msg.WithCalleeServiceName("testServiceName")
	msg.WithSerializationType(-1)
	msg.WithCompressType(0)
	msg.WithClientReqHead(req)
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		rsp = &Response{}
		msg.WithClientRspHead(rsp)
	}
	go func() {
		time.Sleep(time.Second * 2)
		cancel()
	}()
	_, _ = NewClientTransport().RoundTrip(ctx, []byte(""), transport.WithDialAddress(address))

	// async=true
	ctx, cancel = context.WithDeadline(trpc.BackgroundContext(), time.Now().Add(time.Second))
	req = &Request{
		Message: sarama.ProducerMessage{
			Key:   sarama.ByteEncoder([]byte("key")),
			Value: sarama.ByteEncoder([]byte("value")),
		},
		Async: true,
	}
	trpc.SetMetaData(ctx, string("key"), []byte("value"))
	ctx, msg = codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(fmt.Sprintf("/%s/produce", "testServiceName"))
	msg.WithCalleeServiceName("testServiceName")
	msg.WithSerializationType(-1)
	msg.WithCompressType(0)
	msg.WithClientReqHead(req)
	rsp, ok = msg.ClientRspHead().(*Response)
	if !ok {
		rsp = &Response{}
		msg.WithClientRspHead(rsp)
	}
	go func() {
		time.Sleep(time.Second * 2)
		cancel()
	}()
	_, _ = NewClientTransport().RoundTrip(ctx, []byte(""), transport.WithDialAddress(address))

	assert.Nil(t, nil)
}

func TestRoundTripAsyncBlock(t *testing.T) {
	newAsyncProducer = fakeNewBlockedAsyncProducer
	defer func() {
		newAsyncProducer = sarama.NewAsyncProducer
	}()

	ctx, cancel := context.WithDeadline(trpc.BackgroundContext(), time.Now().Add(time.Second))
	req := &Request{
		Message: sarama.ProducerMessage{
			Key:   sarama.ByteEncoder("key"),
			Value: sarama.ByteEncoder("value"),
		},
		Async: true,
	}
	trpc.SetMetaData(ctx, "key", []byte("value"))

	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/produce", "testServiceName"))
	msg.WithCalleeServiceName("testServiceName")
	msg.WithSerializationType(-1)
	msg.WithCompressType(0)
	msg.WithClientReqHead(req)
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		rsp = &Response{}
		msg.WithClientRspHead(rsp)
	}
	address := "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc"

	// Test context cancel
	newCtx, cancel := context.WithCancel(ctx)
	cancel() // make sure the context is canceled
	_, err := NewClientTransport().RoundTrip(newCtx, []byte(""), transport.WithDialAddress(address))
	if assert.IsType(t, &errs.Error{}, err) {
		assert.Equal(t, errs.RetClientCanceled, errs.Code(err))
	}

	// Test context deadline
	newCtx, cancel = context.WithDeadline(ctx, time.Now().Add(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond) // make sure the context is expired
	defer cancel()

	_, err = NewClientTransport().RoundTrip(newCtx, []byte(""), transport.WithDialAddress(address))
	if assert.IsType(t, &errs.Error{}, err) {
		assert.Equal(t, errs.RetClientTimeout, errs.Code(err))
	}

	req.Async = false
	address += "&async=1"
	_, err = NewClientTransport().RoundTrip(ctx, []byte(""), transport.WithDialAddress(address))
	if assert.IsType(t, &errs.Error{}, err) {
		assert.Equal(t, errs.RetClientTimeout, errs.Code(err))
	}
}

func TestGetProducer(t *testing.T) {
	newSyncProducer = fakeNewSyncProducer
	newAsyncProducer = fakeNewAsyncProducer
	defer func() {
		newSyncProducer = sarama.NewSyncProducer
		newAsyncProducer = sarama.NewAsyncProducer
	}()

	address := "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest"
	clientTransport := ClientTransport{
		producers:          map[string]*Producer{},
		producersLock:      sync.RWMutex{},
		asyncProducers:     map[string]*Producer{},
		asyncProducersLock: sync.RWMutex{},
	}
	_, _ = clientTransport.GetProducer(address, time.Second)

	address = "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&async=1"
	_, _ = clientTransport.GetProducer(address, time.Second)
	address = "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&mechanism=PLAIN&user=ckafka_id#username&password=1234&async=1"
	_, err := clientTransport.GetProducer(address, time.Second)
	assert.Nil(t, err)
	address = "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&mechanism=xxxxxx&user=ckafka_id#username&password=1234&async=1"
	_, err = clientTransport.GetProducer(address, time.Second)
	assert.EqualError(t, err, "type:framework, code:131, msg:kafka scram_client.config failed,x.mechanism unknown(xxxxxx)")
	address = "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&mechanism=PLAIN&user=ckafka_id#username&password=1234"
	_, err = clientTransport.GetProducer(address, time.Second)
	assert.Nil(t, err)
	address = "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&mechanism=xxxxxx&user=ckafka_id#username&password=1234"
	_, err = clientTransport.GetProducer(address, time.Second)
	assert.EqualError(t, err, "type:framework, code:131, msg:kafka scram_client.config failed,x.mechanism unknown(xxxxxx)")
	assert.Nil(t, nil)
}

func TestGetAsyncProducer(t *testing.T) {
	newSyncProducer = fakeNewSyncProducer
	newAsyncProducer = fakeNewAsyncProducer
	defer func() {
		newSyncProducer = sarama.NewSyncProducer
		newAsyncProducer = sarama.NewAsyncProducer
	}()

	address := "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&maxRetry=2&retryInterval=1000"
	clientTransport := ClientTransport{
		producers:          map[string]*Producer{},
		producersLock:      sync.RWMutex{},
		asyncProducers:     map[string]*Producer{},
		asyncProducersLock: sync.RWMutex{},
	}
	_, _ = clientTransport.GetAsyncProducer(address, 2*time.Second)
	address = "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&mechanism=PLAIN&user=ckafka_id#username&password=1234"
	_, err := clientTransport.GetAsyncProducer(address, 2*time.Second)
	assert.Nil(t, err)
	address = "test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest&mechanism=xxxxxx&user=ckafka_id#username&password=1234"
	_, err = clientTransport.GetAsyncProducer(address, 2*time.Second)
	assert.EqualError(t, err, "type:framework, code:131, msg:kafka scram_client.config failed,x.mechanism unknown(xxxxxx)")
	assert.Nil(t, nil)
}
