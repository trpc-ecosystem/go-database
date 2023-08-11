package timer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/server"
)

// TestTimerNetwork test parse timer network config
func TestTimerNetwork(t *testing.T) {
	s := &DefaultScheduler{}
	RegisterScheduler("sss", s)
	m := &mockScheduler{}
	RegisterScheduler("mock", m)

	var tests = []struct {
		Network string
		Conf    Config
		Err     error
	}{
		{"0 */1 * * *",
			Config{StartAtOnce: false, Scheduler: &DefaultScheduler{}, HoldTime: time.Second}, nil},
		{"0 */1 * * * ?",
			Config{StartAtOnce: false, Scheduler: &DefaultScheduler{}, HoldTime: time.Second}, nil},
		{"0 */1 * * * ?startAtOnce=1",
			Config{StartAtOnce: true, Scheduler: &DefaultScheduler{}, HoldTime: time.Second}, nil},
		{"0 */1 * * * *?startAtOnce=1&scheduler=sss",
			Config{StartAtOnce: true, Scheduler: s, HoldTime: time.Second}, nil},
		{"0 */1 * * * *?startAtOnce=1&scheduler=sss&holdTime=2",
			Config{StartAtOnce: true, Scheduler: s, HoldTime: 2 * time.Second}, nil},
		{"0 */1 * * * ?&",
			Config{StartAtOnce: false, Scheduler: &DefaultScheduler{}, HoldTime: time.Second}, ErrInvalidNetwork},
		{"0 */1 * * * *?startAtOnce=1&scheduler=sss&holdTime===2",
			Config{StartAtOnce: true, Scheduler: s, HoldTime: 2 * time.Second}, ErrInvalidHoldTime},
		{"0 */1 * * * *?startAtOnce=1&scheduler=xxx",
			Config{StartAtOnce: true, Scheduler: s, HoldTime: time.Second}, ErrInvalidScheduler},
		{"0 */1 * * * *?test=1",
			Config{StartAtOnce: true, Scheduler: s, HoldTime: time.Second}, ErrInvalidNetwork},
	}

	for _, tt := range tests {
		got, err := DefaultServerTransport.(*serverTransport).parseNetwork(tt.Network)
		if tt.Err == nil {
			assert.Nil(t, err)
			assert.Equal(t, tt.Conf.Scheduler, got.Scheduler)
			assert.Equal(t, tt.Conf.HoldTime, got.HoldTime)
			assert.Equal(t, tt.Conf.StartAtOnce, got.StartAtOnce)
			continue
		}
		assert.Equal(t, tt.Err, err)
	}
}

func TestDefaultScheduler(t *testing.T) {
	s := &DefaultScheduler{}
	var (
		serviceName = "testing"
		holdTime    = time.Second
	)
	var tests = []struct {
		node string
	}{
		{"1"},
		{"node"},
		{"@"},
	}
	for _, tt := range tests {
		got, err := s.Schedule(serviceName, tt.node, holdTime)
		assert.Nil(t, err)
		assert.Equal(t, tt.node, got)
	}
}

func TestRegisterHandlerService(t *testing.T) {
	// mock
	got := false
	// action
	{
		service := server.New(
			server.WithProtocol("timer"),
			server.WithNetwork("*/1 * * * * ?"),
		)
		done := make(chan struct{})
		RegisterHandlerService(service, func(ctx context.Context) error {
			got = true
			close(done)
			return nil
		})
		go func() {
			_ = service.Serve()
		}()
		<-done
		service.Close(nil)
	}

	// assert
	assert.True(t, got)
}

func TestScheduleFailed(t *testing.T) {
	// mock
	got := false
	// action
	m := &mockScheduler{}
	RegisterScheduler("m", m)
	{
		service := server.New(
			server.WithProtocol("timer"),
			server.WithNetwork("*/1 * * * * ?startAtOnce=1&scheduler=m"),
		)
		done := make(chan struct{})
		RegisterHandlerService(service, func(ctx context.Context) error {
			got = true
			close(done)
			return nil
		})
		go func() {
			_ = service.Serve()
		}()
		go func() {
			<-time.After(time.Second * 3)
			close(done)
		}()
		<-done
		service.Close(nil)
	}

	// assert
	assert.False(t, got)
}

func TestRegisterTimerService(t *testing.T) {
	// mock
	got := false
	// action
	{
		service := server.New(
			server.WithProtocol("timer"),
			server.WithNetwork("*/1 * * * * ?"),
		)
		done := make(chan struct{})
		var h handler
		h = func(ctx context.Context) error {
			got = true
			close(done)
			return nil
		}
		RegisterTimerService(service, h)
		go func() {
			_ = service.Serve()
		}()
		<-done
		service.Close(nil)
	}
	// assert
	assert.True(t, got)
}

func TestSetStartAtOnce(t *testing.T) {
	// mock
	DefaultServerTransport.(*serverTransport).opts.StartAtOnce = false
	// action
	SetStartAtOnce()
	// assert
	assert.True(t, DefaultServerTransport.(*serverTransport).opts.StartAtOnce)
}

func TestServerTransportOption(t *testing.T) {
	// mock
	scheduler := &DefaultScheduler{}
	DefaultServerTransport.(*serverTransport).opts.StartAtOnce = false
	DefaultServerTransport.(*serverTransport).opts.Scheduler = nil

	// action
	DefaultServerTransport = NewServerTransport(
		WithScheduler(scheduler),
		StartAtOnce(),
	)

	// assert
	assert.True(t, DefaultServerTransport.(*serverTransport).opts.StartAtOnce)
	assert.Equal(t, DefaultServerTransport.(*serverTransport).opts.Scheduler, scheduler)
}

func TestCodec(t *testing.T) {
	svrCodec := &ServerCodec{}
	msg := codec.Message(context.TODO())
	buf := []byte("body")
	// check decode
	{
		rspbuf, err := svrCodec.Decode(msg, buf)
		assert.Nil(t, err)
		assert.Empty(t, rspbuf)
	}
	// check encode
	{
		rspbuf, err := svrCodec.Encode(msg, buf)
		assert.Nil(t, err)
		assert.Empty(t, rspbuf)
	}

	// check service error transfer
	{
		msg.WithServerRspErr(errors.New("testing"))
		rspbuf, err := svrCodec.Encode(msg, buf)
		assert.NotNil(t, err)
		assert.Empty(t, rspbuf)
	}
}

func TestTimerHandlerFailed(t *testing.T) {
	ctx := context.Background()

	{
		f := func(reqbody interface{}) (filter.ServerChain, error) {
			return nil, nil
		}
		rsp, err := TimerHandler(&mockHandler{}, ctx, f)
		assert.NotNil(t, err)
		assert.Nil(t, rsp)
	}

	{
		f := func(reqbody interface{}) (filter.ServerChain, error) {
			return nil, errors.New("invalid chain")
		}
		rsp, err := TimerHandler(nil, ctx, f)
		assert.NotNil(t, err)
		assert.Nil(t, rsp)
	}
}

type mockHandler struct {
}

func (h *mockHandler) Handle(ctx context.Context) error {
	return errors.New("handle failed")
}

type mockScheduler struct {
}

func (s *mockScheduler) Schedule(serviceName string,
	newNode string, holdTime time.Duration) (nowNode string, err error) {
	return "", errors.New("schedule failed")
}
