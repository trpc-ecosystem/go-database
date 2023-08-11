package kafka

import (
	"context"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	transport.RegisterClientTransport("kafka", DefaultClientTransport)
}

// ClientTransport implements the trpc.ClientTransport interface and encapsulate the producer
type ClientTransport struct {
	opts               *transport.ClientTransportOptions
	producers          map[string]*Producer
	producersLock      sync.RWMutex
	asyncProducers     map[string]*Producer
	asyncProducersLock sync.RWMutex
}

// DefaultClientTransport default client kafka transport
var DefaultClientTransport = NewClientTransport()

// AsyncProducerErrorCallback asynchronous production failure callback,
// the default implementation only prints the error log,
// the user can rewrite the callback function to achieve sending error capture.
var AsyncProducerErrorCallback = func(err error, topic string, key, value []byte, headers []sarama.RecordHeader) {
	log.Errorf("asyncProduce failed. topic:%s, key:%s, value:%s. err:%v", topic, key,
		value, err)
}

// AsyncProducerSuccCallback asynchronous production success callback,
// no processing is done by default, the user can rewrite the callback function to achieve sending success capture.
var AsyncProducerSuccCallback = func(topic string, key, value []byte, headers []sarama.RecordHeader) {}

var (
	newSyncProducer  = sarama.NewSyncProducer
	newAsyncProducer = sarama.NewAsyncProducer
)

// NewClientTransport build kafka transport
func NewClientTransport(opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}
	// Write the incoming func option into the opts field
	for _, o := range opt {
		o(opts)
	}
	return &ClientTransport{
		opts:           opts,
		producers:      map[string]*Producer{},
		asyncProducers: map[string]*Producer{},
	}
}

// RoundTrip send and receive kafka packets,
// return the kafka response and put it in ctx, there is no need to return rspbuf
func (ct *ClientTransport) RoundTrip(
	ctx context.Context, _ []byte, callOpts ...transport.RoundTripOption,
) ([]byte, error) {
	msg := codec.Message(ctx)
	req, ok := msg.ClientReqHead().(*Request)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"kafka client transport: ReqHead should be type of *kafka.Request")
	}
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"kafka client transport: RspHead should be type of *kafka.Response")
	}
	opts := &transport.RoundTripOptions{}
	for _, o := range callOpts {
		o(opts)
	}
	if req.Async {
		producer, err := ct.GetAsyncProducer(opts.Address, Timeout)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr,
				"kafka client transport GetAsyncProducer: "+err.Error())
		}
		if req.Message.Topic == "" { // Prioritize the topic passed in as a parameter
			if producer.topic == "" {
				return nil, errs.NewFrameError(errs.RetClientNetErr, "kafka client transport empty topic")
			}
			req.Message.Topic = producer.topic
		}
		if producer.trpcMeta {
			req.Message.Headers = setSaramaHeader(req, msg.ClientMetaData())
		}

		if err = asyncProduceMessage(ctx, producer.asyncProducer.Input(), &req.Message); err != nil {
			return nil, err
		}
	} else {
		producer, err := ct.GetProducer(opts.Address, Timeout)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr, "kafka client transport GetProducer:"+err.Error())
		}
		if req.Message.Topic == "" { // Prioritize the topic passed in by the parameter
			if producer.topic == "" {
				return nil, errs.NewFrameError(errs.RetClientNetErr, "kafka client transport empty topic")
			}
			req.Message.Topic = producer.topic
		}
		if producer.trpcMeta {
			req.Message.Headers = setSaramaHeader(req, msg.ClientMetaData())
		}
		if producer.async { // compatible with old sendmessage logic
			req.Async = true
			if err = asyncProduceMessage(ctx, producer.asyncProducer.Input(), &req.Message); err != nil {
				return nil, err
			}
		} else {
			rsp.Partition, rsp.Offset, err = producer.syncProducer.SendMessage(&req.Message)
			if err != nil {
				return nil, errs.NewFrameError(errs.RetClientNetErr, "kafka client transport SendMessage: "+err.Error())
			}
		}
	}
	return nil, nil
}

// asyncProduceMessage Try to produce messages asynchronously and handle error scenarios
func asyncProduceMessage(ctx context.Context, ch chan<- *sarama.ProducerMessage, m *sarama.ProducerMessage) error {
	select {
	case ch <- m:
		return nil
	case <-ctx.Done():
		// Clearly return the RetClientCanceled error code
		if ctx.Err() == context.Canceled {
			return errs.WrapFrameError(ctx.Err(), errs.RetClientCanceled,
				"kafka client transport select: async producer context canceled")
		}
		return errs.WrapFrameError(ctx.Err(), errs.RetClientTimeout,
			"kafka client transport select: async producer context deadline exceeded")
	}
}

// GetProducer get producer logic
func (ct *ClientTransport) GetProducer(address string, timeout time.Duration) (*Producer, error) {
	ct.producersLock.RLock()
	producer, ok := ct.producers[address]
	ct.producersLock.RUnlock()
	if ok {
		return producer, nil
	}
	ct.producersLock.Lock()
	defer ct.producersLock.Unlock()
	producer, ok = ct.producers[address]
	if ok {
		return producer, nil
	}
	userConfig, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	producer, err = ct.configProducer(userConfig, timeout)
	if err != nil {
		return nil, err
	}
	ct.producers[address] = producer
	return producer, nil
}

func (ct *ClientTransport) configProducer(userConfig *UserConfig, timeout time.Duration) (*Producer, error) {
	config := newSaramaConfig(userConfig, timeout)
	if userConfig.Async == 1 {
		if err := userConfig.ScramClient.config(config); err != nil {
			return nil, err
		}
		p, err := newAsyncProducer(userConfig.Brokers, config)
		if err != nil {
			return nil, err
		}
		go ct.AsyncProduce(p)
		return &Producer{
			async:         true,
			asyncProducer: p,
			topic:         userConfig.Topic,
			trpcMeta:      userConfig.TrpcMeta,
		}, nil
	}

	config.Producer.RequiredAcks = userConfig.RequiredAcks
	if err := userConfig.ScramClient.config(config); err != nil {
		return nil, err
	}
	p, err := newSyncProducer(userConfig.Brokers, config)
	if err != nil {
		return nil, err
	}
	return &Producer{
		syncProducer: p,
		topic:        userConfig.Topic,
		trpcMeta:     userConfig.TrpcMeta,
	}, nil
}

// GetAsyncProducer get an asynchronous producer
// and start an asynchronous coroutine to process production data and messages
func (ct *ClientTransport) GetAsyncProducer(address string, timeout time.Duration) (*Producer, error) {
	ct.asyncProducersLock.RLock()
	producer, ok := ct.asyncProducers[address]
	ct.asyncProducersLock.RUnlock()
	if ok {
		return producer, nil
	}

	ct.asyncProducersLock.Lock()
	defer ct.asyncProducersLock.Unlock()
	producer, ok = ct.asyncProducers[address]
	if ok {
		return producer, nil
	}

	userConfig, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}

	config := newSaramaConfig(userConfig, timeout)
	err = userConfig.ScramClient.config(config)
	if err != nil {
		return nil, err
	}
	p, err := newAsyncProducer(userConfig.Brokers, config)
	if err != nil {
		return nil, err
	}
	go ct.AsyncProduce(p)
	producer = &Producer{
		async:         true,
		asyncProducer: p,
		topic:         userConfig.Topic,
		trpcMeta:      userConfig.TrpcMeta,
	}
	ct.asyncProducers[address] = producer
	return producer, nil
}

// AsyncProduce produce and process captured messages asynchronously
func (ct *ClientTransport) AsyncProduce(producer sarama.AsyncProducer) {
	for {
		select {
		case errMsg := <-producer.Errors():
			if AsyncProducerErrorCallback != nil {
				key, _ := errMsg.Msg.Key.Encode()
				value, _ := errMsg.Msg.Value.Encode()
				AsyncProducerErrorCallback(errMsg.Err, errMsg.Msg.Topic, key, value, errMsg.Msg.Headers)
			}
		case succMsg := <-producer.Successes():
			if AsyncProducerSuccCallback != nil {
				key, _ := succMsg.Key.Encode()
				value, _ := succMsg.Value.Encode()
				AsyncProducerSuccCallback(succMsg.Topic, key, value, succMsg.Headers)
			}
		}
	}
}

func newSaramaConfig(userConfig *UserConfig, timeout time.Duration) *sarama.Config {
	config := sarama.NewConfig()
	config.Version = userConfig.Version
	config.ClientID = userConfig.ClientID
	config.Producer.Return.Successes = userConfig.ReturnSuccesses
	config.Producer.Partitioner = userConfig.Partitioner
	config.Producer.MaxMessageBytes = userConfig.MaxMessageBytes
	config.Producer.Flush.Messages = userConfig.FlushMessages
	config.Producer.Flush.MaxMessages = userConfig.FlushMaxMessages
	config.Producer.Flush.Bytes = userConfig.FlushBytes
	config.Producer.Flush.Frequency = userConfig.FlushFrequency
	config.Producer.Compression = userConfig.Compression
	config.Producer.Retry.Max = userConfig.ProducerRetry.Max
	config.Producer.Retry.Backoff = userConfig.ProducerRetry.RetryInterval
	config.Producer.Idempotent = userConfig.Idempotent

	if timeout > 0 {
		config.Net.DialTimeout = timeout
		config.Net.ReadTimeout = timeout
		config.Net.WriteTimeout = timeout
		config.Producer.Timeout = timeout
		config.Metadata.Timeout = timeout
	}
	return config
}

// setSaramaHeader transparently transmit trpc meta data to sarama header
func setSaramaHeader(req *Request, metaData codec.MetaData) []sarama.RecordHeader {
	for k, v := range metaData {
		h := sarama.RecordHeader{
			Key:   []byte(k),
			Value: v,
		}
		req.Message.Headers = append(req.Message.Headers, h)
	}
	return req.Message.Headers
}
