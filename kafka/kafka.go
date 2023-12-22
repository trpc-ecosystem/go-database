/*
Package kafka encapsulated from github.com/IBM/sarama
Producer sending through trpc.Client
Implement Consumer logic through trpc.Service
*/
package kafka

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"golang.org/x/time/rate"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"
	dsn "trpc.group/trpc-go/trpc-selector-dsn"
)

func init() {
	selector.Register("kafka", dsn.DefaultSelector)
}

// ContinueWithoutCommitError whether to continue to consume messages without commit ack
// Scenes:
// When producing a message, a message body may exceed the limit of Kafka,
// so the original message body will be split into multiple byte packets,
// encapsulated into a Kafka message body, and delivered.
// Then, when consuming messages,
// you need to wait for all subcontracted messages to be consumed before starting business logic processing.
// When the consumer's Handle method or msg.ServerRspErr returns this error,
// It means that you want to continue to consume messages without commit ack and not treat them as errors.
var ContinueWithoutCommitError = &errs.Error{
	Type: errs.ErrorTypeBusiness,
	Code: errs.RetUnknown,
	Msg:  "Error:Continue to consume message without committing ack",
}

// Timeout is the global timeout configuration of Kafka, the default is 2s, and users can modify it if necessary.
var Timeout = 2 * time.Second

// IsCWCError Check if it is a ContinueWithoutCommitError error
// CWC:Continue Without Commit
func IsCWCError(err error) bool {
	return err == ContinueWithoutCommitError
}

var (
	serviceCloseError = errs.NewFrameError(errs.RetServerSystemErr, "kafka consumer service close")
	sessionCloseError = errs.NewFrameError(errs.RetServerSystemErr, "kafka consumer group session close")
	messageCloseError = errs.NewFrameError(errs.RetServerSystemErr, "kafka consumer group claim message close")
)

// Producer encapsulation Producer information
type Producer struct {
	topic         string
	async         bool
	asyncProducer sarama.AsyncProducer
	syncProducer  sarama.SyncProducer
	trpcMeta      bool
}

// singleConsumerHandler implement the sarama consumer interface and consume data item by item.
type singleConsumerHandler struct {
	opts          *transport.ListenServeOptions
	ctx           context.Context
	retryMax      int           // Maximum number of retries
	retryInterval time.Duration // retry interval
	trpcMeta      bool          // Whether to pass meta information
	limiter       *rate.Limiter // current limiter
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *singleConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	if h.retryInterval == 0 {
		h.retryInterval = time.Millisecond
	}
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *singleConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *singleConsumerHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		if err := h.limiter.Wait(h.ctx); err != nil {
			if err == h.ctx.Err() {
				return serviceCloseError
			}
			return fmt.Errorf("kafka server transport: limiter error: %w", err)
		}
		select {
		case <-h.ctx.Done(): // determine whether the service is over
			return serviceCloseError
		case <-sess.Context().Done(): // determine whether the session is over
			return sessionCloseError
		case msg, ok := <-claim.Messages(): // listen for messages
			if !ok { // msg close  and return
				return messageCloseError
			}
			h.retryConsumeAndMark(sess, msg, claim)
		}
	}
}

// RawSaramaContext set sarama ConsumerGroupSession and ConsumerGroupClaim
// This structure is exported for the convenience of users to implement monitoring,
// the content provided is only for reading, calling any write method is an undefined behavior
type RawSaramaContext struct {
	Session sarama.ConsumerGroupSession
	Claim   sarama.ConsumerGroupClaim
}

type rawSaramaContextKey struct{}

func withRawSaramaContext(ctx context.Context, raw *RawSaramaContext) context.Context {
	return context.WithValue(ctx, rawSaramaContextKey{}, raw)
}

// GetRawSaramaContext get sarama raw context information, include ConsumerGroupSession and ConsumerGroupClaim
// The retrieved context should only use read methods, using any write methods is undefined behavior
func GetRawSaramaContext(ctx context.Context) (*RawSaramaContext, bool) {
	rawContext, ok := ctx.Value(rawSaramaContextKey{}).(*RawSaramaContext)
	return rawContext, ok
}

func (h *singleConsumerHandler) retryConsumeAndMark(
	session sarama.ConsumerGroupSession,
	m *sarama.ConsumerMessage,
	claim sarama.ConsumerGroupClaim,
) {
	retryNum := 0
	for {
		ctx, trpcMsg := genTRPCMessage(m, h.opts.ServiceName, m.Topic, h.trpcMeta)
		ctx = withRawSaramaContext(ctx, &RawSaramaContext{Session: session, Claim: claim})
		// Hand it over to the trpc framework
		_, err := h.opts.Handler.Handle(ctx, nil)
		rspErr := trpcMsg.ServerRspErr()
		// If the consumption does not fail, jump out of the loop, mark message and continue to consume subsequent data
		if err == nil && rspErr == nil {
			break
		}

		// error handling logic
		msgInfo := fmt.Sprintf("%s:%d:%d", m.Topic, m.Partition, m.Offset)
		// If it is IsCWCError return directly, do not submit
		if IsCWCError(rspErr) {
			log.WarnContextf(ctx, "kafka consumer handle warn:%v, msg: %+v", rspErr, msgInfo)
			return
		}

		// If processing fails, write error log, number of retries +1
		retryNum++
		log.ErrorContextf(ctx, "kafka consumer msg %s try time %d get fail:%v rspErr:%v, ", msgInfo, retryNum, err, rspErr)

		// If the maximum number of retries is exceeded,
		// end the loop mark message and continue to consume subsequent data
		if h.retryMax != 0 && retryNum > h.retryMax {
			break
		}

		// If you need to retry, wait for a while and execute again
		t := time.NewTimer(h.retryInterval)
		select {
		case <-h.ctx.Done():
			return
		case <-session.Context().Done():
			return
		case <-t.C:
			// retry
		}
	}
	session.MarkMessage(m, "")
}

// batchConsumerHandler bulk consumer
type batchConsumerHandler struct {
	opts *transport.ListenServeOptions
	ctx  context.Context

	maxNum        int // batch maximum quantity
	flushInterval time.Duration
	retryMax      int // failed max retries
	retryInterval time.Duration
	trpcMeta      bool          // Whether to pass meta information
	limiter       *rate.Limiter // current limiter
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *batchConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	if h.retryInterval == 0 {
		h.retryInterval = time.Millisecond
	}
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *batchConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim batch consumer
// Consumption is triggered when maxNum messages are satisfied,
// and consumption is also triggered when the refresh interval is up,
// so as to avoid blocking consumption when the message flow is uneven.
// If the business consumption fails, the entire batch is retried, and only failed messages are not supported
func (h *batchConsumerHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	msgArray := make([]*sarama.ConsumerMessage, h.maxNum)
	idx := 0

	ticker := time.NewTicker(h.flushInterval)
	defer ticker.Stop()

	for {
		if err := h.limiter.Wait(h.ctx); err != nil {
			if err == h.ctx.Err() {
				return serviceCloseError
			}
			return fmt.Errorf("kafka server transport: limiter error: %w", err)
		}
		select {
		case <-h.ctx.Done(): // determine whether the service is over
			return serviceCloseError
		case <-session.Context().Done(): // determine whether the session is over
			return sessionCloseError
		case msg, ok := <-claim.Messages():
			if !ok { // return if message is closed
				return messageCloseError
			}

			msgArray[idx] = msg
			idx++

			if idx >= h.maxNum { // The data has reached the cache maximum
				// Make a deep copy, otherwise msgArray will be overwritten during downstream asynchronous processing
				handleMsg := make([]*sarama.ConsumerMessage, len(msgArray))
				copy(handleMsg, msgArray)
				h.retryConsumeAndMark(session, claim, handleMsg...)
				idx = 0
			}
		case <-ticker.C:
			if idx > 0 {
				// Make a deep copy, otherwise msgArray will be overwritten during downstream asynchronous processing
				handleMsg := make([]*sarama.ConsumerMessage, idx)
				copy(handleMsg, msgArray[:idx])
				h.retryConsumeAndMark(session, claim, handleMsg...)
				idx = 0
			}
		}
	}
}

func (h *batchConsumerHandler) retryConsumeAndMark(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	msgs ...*sarama.ConsumerMessage,
) {
	retryNum := 0
	for {
		ctx, trpcMsg := genTRPCMessage(msgs, h.opts.ServiceName, msgs[0].Topic, h.trpcMeta)
		ctx = withRawSaramaContext(ctx, &RawSaramaContext{Session: session, Claim: claim})
		// hand it over to the trpc framework
		_, err := h.opts.Handler.Handle(ctx, nil)
		rspErr := trpcMsg.ServerRspErr()
		// if the consumption does not fail, jump out of the loop, mark message and continue to consume subsequent data
		if err == nil && rspErr == nil {
			break
		}

		// if processing fails, write error log, retry times +1
		retryNum++
		offset := make([]string, len(msgs))
		for i, v := range msgs {
			offset[i] = strconv.Itoa(int(v.Offset))
		}
		msg := msgs[0]
		info := fmt.Sprintf("topic: %s partition: %d offset: %s", msg.Topic, msg.Partition, strings.Join(offset, ","))
		log.ErrorContextf(ctx, "kafka consumer %s try number %d err: %v  msgErr: %v, ", info, retryNum, err, rspErr)

		// if the maximum number of retries is exceeded,
		// end the loop mark message and continue to consume subsequent data
		if h.retryMax != 0 && retryNum > h.retryMax {
			break
		}

		// if you need to retry, wait for a while and execute again
		t := time.NewTimer(h.retryInterval)
		select {
		case <-h.ctx.Done():
			return
		case <-session.Context().Done():
			return
		case <-t.C:
			// retry
		}
	}
	session.MarkMessage(msgs[len(msgs)-1], "")
}

// genTRPCMessage generate a new TRPC message, save the head, and set the upstream and downstream service names
func genTRPCMessage(reqHead interface{}, serviceName, topic string, trpcMeta bool) (context.Context, codec.Msg) {
	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithServerReqHead(reqHead)
	msg.WithCompressType(codec.CompressTypeNoop) // do not decompress
	msg.WithCallerServiceName("trpc.kafka.noserver.noservice")
	msg.WithCallerMethod(topic)
	msg.WithCalleeServiceName(serviceName)
	msg.WithCalleeApp("kafka") // consistent with the old logic
	msg.WithServerRPCName("/trpc.kafka.consumer.service/handle")
	msg.WithCalleeMethod(topic) // modify the called method to topic name
	if trpcMeta {
		m := getMessageHead(reqHead)
		setTRPCMeta(ctx, m) // equal to msg.WithServerMetaData(req.GetTransInfo())
	}
	return ctx, msg
}

// setTRPCMeta sarama header carry data set to trpc meta
func setTRPCMeta(ctx context.Context, hs []*sarama.RecordHeader) {
	if hs == nil {
		return
	}
	for _, header := range hs {
		trpc.SetMetaData(ctx, string(header.Key), header.Value)
	}
}

// getMessageHead parse a single message from reqHead
func getMessageHead(reqHead interface{}) []*sarama.RecordHeader {
	switch reqHead := reqHead.(type) {
	case *sarama.ConsumerMessage:
		if reqHead != nil {
			return reqHead.Headers
		}
	case []*sarama.ConsumerMessage:
		// for batch consumption, get the metadata of the first message
		if len(reqHead) > 0 && reqHead[0] != nil {
			return reqHead[0].Headers
		}
	}
	// default or slice empty
	return nil
}
