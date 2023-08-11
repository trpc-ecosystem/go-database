package timer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"

	"github.com/robfig/cron"
)

var (
	// ErrInvalidNetwork is a err of invalid network.
	ErrInvalidNetwork = errors.New("invalid network")
	// ErrInvalidHoldTime is a err of invalid holdTime.
	ErrInvalidHoldTime = errors.New("invalid holdTime")
	// ErrInvalidScheduler is a err of invalid scheduler.
	ErrInvalidScheduler = errors.New("invalid scheduler")
)

func init() {
	transport.RegisterServerTransport("timer", DefaultServerTransport)
}

// DefaultServerTransport is the default implement of ServerTransport.
var DefaultServerTransport = NewServerTransport()

// NewServerTransport creates a new ServerTransport.
func NewServerTransport(opt ...ServerTransportOption) transport.ServerTransport {
	// the default options.
	opts := &ServerTransportOptions{
		StartAtOnce: false,
		Scheduler:   &DefaultScheduler{},
	}

	for _, o := range opt {
		o(opts)
	}

	return &serverTransport{opts: opts}
}

// SetStartAtOnce sets the default transport to run the handle function when it starts.
func SetStartAtOnce() {
	DefaultServerTransport.(*serverTransport).opts.StartAtOnce = true
}

// serverTransport is the server transport implementation, including tcp udp serving.
type serverTransport struct {
	opts *ServerTransportOptions
}

var serverName = path.Base(os.Args[0])

// ListenAndServe starts the listener, and return err if the listener fails.
func (s *serverTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) (err error) {
	lsopts := &transport.ListenServeOptions{}
	for _, opt := range opts {
		opt(lsopts)
	}

	pid := os.Getpid()
	addr, _ := net.ResolveTCPAddr("tcp4", lsopts.Address)
	c := cron.New()

	conf, err := s.parseNetwork(lsopts.Network)
	if err != nil {
		return err
	}

	handler := func() error {
		node := fmt.Sprintf("%s_%d_%d", lsopts.Address, pid, time.Now().Unix())
		if nowNode, err := conf.Scheduler.Schedule(conf.ServiceName, node, conf.HoldTime); err != nil {
			log.TraceContextf(ctx, "Scheduler preempt fail:%v, current preempted node is %s", err, nowNode)
			return nil
		}
		// create a new empty message, and save it to ctx.
		ctx, msg := codec.WithNewMessage(ctx)
		// save LocalAddr and RemoteAddr to ctx.
		msg.WithRemoteAddr(addr)
		msg.WithLocalAddr(addr)
		// without decompress.
		msg.WithCompressType(codec.CompressTypeNoop)
		select {
		case <-ctx.Done():
			c.Stop()
			return context.DeadlineExceeded
		default:
		}

		if _, err := lsopts.Handler.Handle(ctx, nil); err != nil {
			return err
		}
		return nil
	}
	if conf.StartAtOnce {
		if err := handler(); err != nil {
			return err
		}
	}

	err = c.AddFunc(conf.Spec, func() {
		if err := handler(); err != nil {
			log.TraceContextf(ctx, "timer handle fail: %v", err)
		}
	})

	if err != nil {
		return err
	}

	c.Start()

	return nil
}

// Config is the configuration for the timer.
type Config struct {
	Scheduler   Scheduler     // distributed timer.
	StartAtOnce bool          //
	Spec        string        //
	ServiceName string        // the current service rpc name, support multiple services.
	HoldTime    time.Duration // the time when the current node wants to hold the task, and the unit is seconds.
}

// parseNetwork parses the time network config.
// 0 */1 * * * ?
// 0 */1 * * * *?startAtOnce=1&scheduler=xxx&holdTime=1
func (s *serverTransport) parseNetwork(network string) (*Config, error) {
	config := &Config{
		StartAtOnce: s.opts.StartAtOnce,
		Scheduler:   s.opts.Scheduler,
		ServiceName: getServiceName(serverName, "default"),
		HoldTime:    time.Second,
	}
	tokens := splitNetwork(network)
	// the network config format similar as: 0 */1 * * * or 0 */1 * * * ?
	if len(tokens) != 2 || tokens[1] == "" {
		config.Spec = network // set network to config spec.
		return config, nil
	}

	config.Spec = tokens[0]
	tokens = strings.Split(tokens[1], "&")
	for _, val := range tokens {
		vals := strings.SplitN(val, "=", 2)
		if len(vals) != 2 {
			return nil, ErrInvalidNetwork
		}
		switch vals[0] {
		case "startAtOnce":
			if vals[1] == "1" {
				config.StartAtOnce = true
			}
		case "holdTime":
			t, err := strconv.Atoi(vals[1])
			if err != nil {
				return nil, ErrInvalidHoldTime
			}
			config.HoldTime = time.Duration(t) * time.Second

		case "scheduler":
			s, ok := schedulers[vals[1]]
			if !ok {
				return nil, ErrInvalidScheduler
			}
			config.Scheduler = s
			config.ServiceName = getServiceName(serverName, vals[1])

		default:
			return nil, ErrInvalidNetwork
		}
	}
	return config, nil
}

func getServiceName(serverName string, scheduleName string) string {
	return fmt.Sprintf("trpc.timer.%s.%s", serverName, scheduleName)
}

func splitNetwork(network string) []string {
	index := strings.LastIndex(network, "?")
	if index == -1 {
		return []string{network}
	}

	return []string{network[0:index], network[index+1:]}
}
