package timer

import (
	"context"

	"trpc.group/trpc-go/trpc-go/server"
)

// RegisterTimerService registers timer service.
func RegisterTimerService(s server.Service, svr Handler) {
	s.Register(&ServiceDesc, svr)
}

// RegisterHandlerService registers timer service.
func RegisterHandlerService(s server.Service, handle func(context.Context) error) {
	s.Register(&ServiceDesc, handler(handle))
}

// Handler the timer handler interface.
type Handler interface {
	Handle(ctx context.Context) error
}

// TimerHandler is the timer service handler.
func TimerHandler(svr interface{}, ctx context.Context, f server.FilterFunc) (rspbody interface{}, err error) {
	filters, err := f(nil)
	if err != nil {
		return nil, err
	}

	handleFunc := func(ctx context.Context, reqbody interface{}) (interface{}, error) {
		return nil, svr.(Handler).Handle(ctx)
	}

	rspbody, err = filters.Filter(ctx, nil, handleFunc)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ServiceDesc descriptor for server.RegisterService.
var ServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.timer.local.service",
	HandlerType: (*Handler)(nil),
	Methods: []server.Method{
		{
			Name: "/trpc.timer.server.service/handle",
			Func: TimerHandler,
		},
	},
}

type handler func(context.Context) error

// Handle change func to interface.
func (h handler) Handle(ctx context.Context) error {
	return h(ctx)
}
