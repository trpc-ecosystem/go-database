package kafka

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"
)

type testHandler struct {
	runTime map[string]int
	err     error
	rspErr  error
}

func (th *testHandler) count(message *sarama.ConsumerMessage) {
	if len(th.runTime) == 0 {
		th.runTime = map[string]int{}
	}
	th.runTime[fmt.Sprintf("%s:%d:%d", message.Topic, message.Partition, message.Offset)]++
}

func (th *testHandler) Handle(ctx context.Context, req []byte) ([]byte, error) {
	msg := codec.Message(ctx)
	if message, ok := msg.ServerReqHead().(*sarama.ConsumerMessage); ok {
		th.count(message)
	}
	if messages, ok := msg.ServerReqHead().([]*sarama.ConsumerMessage); ok {
		for _, message := range messages {
			th.count(message)
		}
	}
	if th.rspErr != nil {
		msg.WithServerRspErr(th.rspErr)
	}
	return []byte{}, th.err
}

type testGroupClaim struct {
	msg chan *sarama.ConsumerMessage
}

func (tgc *testGroupClaim) Topic() string {
	return "test"
}

func (tgc *testGroupClaim) Partition() int32 {
	return 0
}

func (tgc *testGroupClaim) InitialOffset() int64 {
	return 0
}

func (tgc *testGroupClaim) HighWaterMarkOffset() int64 {
	return 0
}

func (tgc *testGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return tgc.msg
}

type testGroup struct{}

func (tg testGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	return nil
}

func (tg testGroup) Errors() <-chan error {
	ec := make(chan error, 1)
	return ec
}

func (tg testGroup) Close() error {
	return nil
}

type testGroupSession struct {
	ctx    context.Context
	cancel context.CancelFunc

	offset int64
}

func (tgs *testGroupSession) Claims() map[string][]int32 {
	return map[string][]int32{}
}

func (tgs *testGroupSession) MemberID() string {
	return "123"
}

func (tgs *testGroupSession) GenerationID() int32 {
	return 0
}

func (tgs *testGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}

func (tgs *testGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}

func (tgs *testGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	tgs.offset = msg.Offset
}

func (tgs *testGroupSession) Context() context.Context {
	return tgs.ctx
}

func (tgs *testGroupSession) Commit() {
}

type testAsyncProducer struct{}

func (tap testAsyncProducer) AsyncClose() {
	return
}

func (tap testAsyncProducer) Close() error {
	return nil
}

func (tap testAsyncProducer) Input() chan<- *sarama.ProducerMessage {
	ret := make(chan *sarama.ProducerMessage, 2)
	return ret
}

func (tap testAsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	ret := make(chan *sarama.ProducerMessage, 2)
	return ret
}

func (tap testAsyncProducer) Errors() <-chan *sarama.ProducerError {
	ret := make(chan *sarama.ProducerError, 2)
	return ret
}

type testBlockedAsyncProducer struct{}

func (t testBlockedAsyncProducer) AsyncClose() {}

func (t testBlockedAsyncProducer) Close() error {
	return nil
}

func (t testBlockedAsyncProducer) Input() chan<- *sarama.ProducerMessage {
	return make(chan *sarama.ProducerMessage) // block forever
}

func (t testBlockedAsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return make(chan *sarama.ProducerMessage) // block forever
}

func (t testBlockedAsyncProducer) Errors() <-chan *sarama.ProducerError {
	return make(chan *sarama.ProducerError) // block forever
}

type testSyncProducer struct{}

func (tsp testSyncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	return 1, 10, nil
}

func (tsp testSyncProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	return nil
}

func (tsp testSyncProducer) Close() error {
	return nil
}

func newMsg(offset int64) *sarama.ConsumerMessage {
	return &sarama.ConsumerMessage{
		Timestamp:      time.Now(),
		BlockTimestamp: time.Now(),
		Key:            []byte("msg key"),
		Value:          []byte("msg value"),
		Topic:          "topic",
		Partition:      0,
		Offset:         offset,
		Headers:        []*sarama.RecordHeader{{Key: []byte("key"), Value: []byte("value")}},
	}
}

func newMsgChan(n int) chan *sarama.ConsumerMessage {
	rsp := make(chan *sarama.ConsumerMessage, n)
	for i := 0; i < n; i++ {
		rsp <- newMsg(int64(i))
	}
	return rsp
}

func newTestSession(timeout time.Duration) *testGroupSession {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	return &testGroupSession{
		ctx:    ctx,
		cancel: cancel,
		offset: 0,
	}
}

// Test single data consumption logic Retry logic
func Test_singleConsumerHandler_retry(t *testing.T) {
	claim := &testGroupClaim{msg: make(chan *sarama.ConsumerMessage, 100)}

	msg := newMsg(int64(rand.Intn(100)))
	msgKey := fmt.Sprintf("topic:0:%d", msg.Offset)
	// Test 0 retries
	tests := []struct {
		name       string
		retryMax   int
		handler    *testHandler
		sTimeout   time.Duration
		wantGte    int
		wantOffset int64
		wantErr    error
	}{
		{
			name:       "retryMax=3WithOk",
			retryMax:   3,
			handler:    &testHandler{},
			sTimeout:   time.Second,
			wantGte:    1,
			wantOffset: msg.Offset,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=0WithOk",
			retryMax:   0,
			handler:    &testHandler{},
			sTimeout:   time.Second,
			wantGte:    1,
			wantOffset: msg.Offset,
			wantErr:    serviceCloseError,
		},
		{
			name:       "ContinueWithoutCommitError",
			retryMax:   0,
			handler:    &testHandler{rspErr: ContinueWithoutCommitError},
			sTimeout:   time.Second,
			wantGte:    1,
			wantOffset: 0,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=-1WithError",
			retryMax:   -1,
			handler:    &testHandler{err: fmt.Errorf("handler error")},
			sTimeout:   time.Second,
			wantGte:    1,
			wantOffset: msg.Offset,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=2WithError",
			retryMax:   2,
			handler:    &testHandler{err: fmt.Errorf("handler error")},
			sTimeout:   time.Second,
			wantGte:    3,
			wantOffset: msg.Offset,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=0WithError",
			retryMax:   0,
			handler:    &testHandler{err: fmt.Errorf("handler error")},
			sTimeout:   time.Second,
			wantGte:    10,
			wantOffset: 0,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=0WithErrorSess",
			retryMax:   0,
			handler:    &testHandler{err: fmt.Errorf("handler error")},
			sTimeout:   time.Millisecond,
			wantGte:    1,
			wantOffset: 0,
			wantErr:    sessionCloseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Millisecond)
			defer cancel()
			consumerHandler := &singleConsumerHandler{
				opts: &transport.ListenServeOptions{
					ServiceName: "trpc.kafka.server.service",
					Handler:     tt.handler,
				},
				ctx:      ctx,
				retryMax: tt.retryMax,
				trpcMeta: true,
				limiter:  newLimiter(nil),
			}

			session := newTestSession(tt.sTimeout)
			claim.msg <- msg
			err := consumerHandler.ConsumeClaim(session, claim)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantOffset, session.offset)
			assert.GreaterOrEqual(t, tt.handler.runTime[msgKey], tt.wantGte, "retry count")
		})
	}
}

// Test single data consumption logic
func Test_singleConsumerHandler(t *testing.T) {
	handlerCtx, handlerCancel := context.WithCancel(context.Background())
	consumerHandler := &singleConsumerHandler{
		opts: &transport.ListenServeOptions{
			ServiceName: "trpc.kafka.server.service",
			Handler:     &testHandler{},
		},
		ctx:      handlerCtx,
		trpcMeta: true,
		limiter:  newLimiter(nil),
	}
	testSession := &testGroupSession{}
	testClaim := &testGroupClaim{msg: make(chan *sarama.ConsumerMessage)}

	_ = consumerHandler.Setup(testSession)
	defer func() {
		_ = consumerHandler.Cleanup(testSession)
	}()

	// test session
	t.Run("session.cancel", func(t *testing.T) {
		testSession.ctx, testSession.cancel = context.WithTimeout(ctx, time.Millisecond)
		err := consumerHandler.ConsumeClaim(testSession, testClaim)
		assert.Equal(t, err, sessionCloseError)
	})

	t.Run("message.cancel", func(t *testing.T) {
		close(testClaim.msg)
		testSession.ctx, testSession.cancel = context.WithTimeout(ctx, time.Millisecond)
		err := consumerHandler.ConsumeClaim(testSession, testClaim)
		assert.Equal(t, err, messageCloseError)
	})

	t.Run("server.cancel", func(t *testing.T) {
		handlerCancel()
		testClaim.msg = make(chan *sarama.ConsumerMessage, 0)
		testSession.ctx, testSession.cancel = context.WithTimeout(ctx, time.Millisecond)
		err := consumerHandler.ConsumeClaim(testSession, testClaim)
		assert.Equal(t, err, serviceCloseError)
	})
}

// Test batch consumption logic
func Test_batchConsumerHandler(t *testing.T) {
	handlerCtx, handlerCancel := context.WithCancel(context.Background())
	th := &testHandler{}
	batchConsumerHandler := &batchConsumerHandler{
		opts: &transport.ListenServeOptions{
			ServiceName: "trpc.kafka.server.service",
			Handler:     th,
		},
		ctx:           handlerCtx,
		maxNum:        10,
		flushInterval: time.Second,
		trpcMeta:      true,
		limiter:       newLimiter(nil),
	}
	testSession := &testGroupSession{}
	testClaim := &testGroupClaim{msg: make(chan *sarama.ConsumerMessage)}

	_ = batchConsumerHandler.Setup(testSession)
	defer func() {
		_ = batchConsumerHandler.Cleanup(testSession)
	}()

	// test session
	t.Run("session.cancel", func(t *testing.T) {
		testSession.ctx, testSession.cancel = context.WithTimeout(ctx, time.Millisecond)
		err := batchConsumerHandler.ConsumeClaim(testSession, testClaim)
		assert.Equal(t, err, sessionCloseError)
	})

	t.Run("message.cancel", func(t *testing.T) {
		close(testClaim.msg)
		testSession.ctx, testSession.cancel = context.WithTimeout(ctx, time.Millisecond)
		err := batchConsumerHandler.ConsumeClaim(testSession, testClaim)
		assert.Equal(t, err, messageCloseError)
	})

	t.Run("server.cancel", func(t *testing.T) {
		handlerCancel()
		testClaim.msg = make(chan *sarama.ConsumerMessage)
		testSession.ctx, testSession.cancel = context.WithTimeout(ctx, time.Millisecond)
		err := batchConsumerHandler.ConsumeClaim(testSession, testClaim)
		assert.Equal(t, err, serviceCloseError)
	})

	t.Run("flushInterval", func(t *testing.T) {
		handlerCtx, handlerCancel = context.WithCancel(context.Background())
		batchConsumerHandler.ctx = handlerCtx
		batchConsumerHandler.flushInterval = time.Millisecond * 10

		testSession.ctx, testSession.cancel = context.WithTimeout(ctx, time.Second)

		testClaim.msg = make(chan *sarama.ConsumerMessage, 1)
		testClaim.msg <- newMsg(1)
		err := batchConsumerHandler.ConsumeClaim(testSession, testClaim)
		assert.Equal(t, err, sessionCloseError)
		assert.Equal(t, len(th.runTime), 1)
	})

	t.Run("moreThanMax", func(t *testing.T) {
		handlerCtx, handlerCancel = context.WithCancel(context.Background())
		batchConsumerHandler.ctx = handlerCtx
		batchConsumerHandler.flushInterval = 2 * time.Second
		batchConsumerHandler.maxNum = 2
		testSession.ctx, testSession.cancel = context.WithTimeout(ctx, time.Second)
		th.runTime = map[string]int{}

		testClaim.msg = make(chan *sarama.ConsumerMessage, 3)
		testClaim.msg <- newMsg(2)
		testClaim.msg <- newMsg(3)
		testClaim.msg <- newMsg(4)
		err := batchConsumerHandler.ConsumeClaim(testSession, testClaim)
		assert.Equal(t, err, sessionCloseError)
		assert.Equal(t, len(th.runTime), 2)
	})
}

func Test_batchConsumerHandler_retry(t *testing.T) {
	tests := []struct {
		name       string
		retryMax   int           // max retry
		handler    *testHandler  // transport handler
		sTimeout   time.Duration // session timeout
		wantGte    int
		wantOffset int64
		wantErr    error
	}{
		{
			name:       "retryMax=3WithOk",
			retryMax:   3,
			handler:    &testHandler{},
			sTimeout:   time.Second,
			wantGte:    1,
			wantOffset: 9,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=0WithOk",
			retryMax:   0,
			handler:    &testHandler{},
			sTimeout:   time.Second,
			wantGte:    1,
			wantOffset: 9,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=-1WithError",
			retryMax:   -1,
			handler:    &testHandler{err: fmt.Errorf("handler error")},
			sTimeout:   time.Second,
			wantGte:    1,
			wantOffset: 9,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=2WithError",
			retryMax:   2,
			handler:    &testHandler{err: fmt.Errorf("handler error")},
			sTimeout:   time.Second,
			wantGte:    1,
			wantOffset: 9,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=0WithError",
			retryMax:   0,
			handler:    &testHandler{err: fmt.Errorf("handler error")},
			sTimeout:   time.Second,
			wantGte:    10,
			wantOffset: 0,
			wantErr:    serviceCloseError,
		},
		{
			name:       "retryMax=0WithError",
			retryMax:   0,
			handler:    &testHandler{err: fmt.Errorf("handler error")},
			sTimeout:   time.Millisecond,
			wantGte:    1,
			wantOffset: 0,
			wantErr:    sessionCloseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Millisecond)
			defer cancel()
			claim := &testGroupClaim{msg: newMsgChan(10)}
			consumerHandler := &batchConsumerHandler{
				opts: &transport.ListenServeOptions{
					ServiceName: "trpc.kafka.server.service",
					Handler:     tt.handler,
				},
				maxNum:        10,
				flushInterval: tt.sTimeout,
				ctx:           ctx,
				retryMax:      tt.retryMax,
				limiter:       newLimiter(nil),
			}
			session := newTestSession(tt.sTimeout)
			err := consumerHandler.ConsumeClaim(session, claim)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantOffset, session.offset)
			assert.GreaterOrEqual(t, tt.handler.runTime["topic:0:9"], tt.wantGte, "retry count")
		})
	}
}

func Test_genTRPCMessage(t *testing.T) {
	t.Run("genTRPCMessageOK", func(t *testing.T) {
		reqHead := make([]*sarama.ConsumerMessage, 0)
		serviceName := "trpc.kafka.server.service"
		reqHead = append(reqHead, newMsg(1))
		reqHead[0].Headers = nil
		ctx, trpcMsg := genTRPCMessage(reqHead, serviceName, "topic", true)
		assert.NotNil(t, ctx)
		assert.NotNil(t, trpcMsg)
	})

	t.Run("genTRPCMessageNil", func(t *testing.T) {
		serviceName := "trpc.kafka.server.service"
		reqHead := newMsg(1)
		reqHead = nil
		ctx, trpcMsg := genTRPCMessage(reqHead, serviceName, "topic", true)
		assert.NotNil(t, ctx)
		assert.NotNil(t, trpcMsg)
	})

	t.Run("genTRPCMessageSliceNil", func(t *testing.T) {
		reqHead := make([]*sarama.ConsumerMessage, 0)
		serviceName := "trpc.kafka.server.service"
		reqHead = nil
		ctx, trpcMsg := genTRPCMessage(reqHead, serviceName, "topic", true)
		assert.NotNil(t, ctx)
		assert.NotNil(t, trpcMsg)
	})

	t.Run("genTRPCMessageSliceNil", func(t *testing.T) {
		reqHead := make([]*sarama.ConsumerMessage, 0)
		serviceName := "trpc.kafka.server.service"
		reqHead = append(reqHead, newMsg(1))
		reqHead[0] = nil
		ctx, trpcMsg := genTRPCMessage(reqHead, serviceName, "topic", true)
		assert.NotNil(t, ctx)
		assert.NotNil(t, trpcMsg)
	})

	t.Run("genTRPCMessageErrorType", func(t *testing.T) {
		reqHead := make([]*int, 0)
		serviceName := "trpc.kafka.server.service"
		reqHead = nil
		ctx, trpcMsg := genTRPCMessage(reqHead, serviceName, "topic", true)
		assert.NotNil(t, ctx)
		assert.NotNil(t, trpcMsg)
	})
}

func Test_RawSaramaContext(t *testing.T) {
	t.Run("RawSaramaContextOK", func(t *testing.T) {
		ctx := context.Background()
		ctx = withRawSaramaContext(ctx, &RawSaramaContext{Claim: &testGroupClaim{}, Session: &testGroupSession{}})
		rawContext, ok := GetRawSaramaContext(ctx)
		assert.True(t, ok)
		assert.NotNil(t, rawContext)
	})

	t.Run("RawSaramaContextNil", func(t *testing.T) {
		ctx := context.Background()
		rawContext, ok := GetRawSaramaContext(ctx)
		assert.False(t, ok)
		assert.Nil(t, rawContext)
	})
}
