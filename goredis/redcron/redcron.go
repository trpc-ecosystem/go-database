// Package redcron is distributed timers.
package redcron

import (
	"context"
	"fmt"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
	cron "github.com/robfig/cron/v3"
	goredis "trpc.group/trpc-go/trpc-database/goredis"
	"trpc.group/trpc-go/trpc-database/goredis/internal/joinfilters"
	pb "trpc.group/trpc-go/trpc-database/goredis/internal/proto"
	"trpc.group/trpc-go/trpc-database/goredis/redcas"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
)

// RedCron is distributed timing task object.
type RedCron struct {
	cmdable redis.Cmdable
	cron    *cron.Cron // Custom cron timer parser.
}

// New creates a new distributed scheduled task object.
func New(cmdable redis.Cmdable, opts ...cron.Option) (*RedCron, error) {
	// Modify the default parser to support second-level timers.
	opts = append([]cron.Option{cron.WithSeconds()}, opts...)
	c := &RedCron{
		cmdable: cmdable,
		cron:    cron.New(opts...),
	}
	return c, nil
}

// AddFunc adds tasks through functions.
// If target does not exist, it means that the scheduled task will not be executed,
// and EntryID=0 will be returned.
func (c *RedCron) AddFunc(name string, f filter.ClientHandleFunc, opts ...Option) (cron.EntryID, error) {
	req := &Request{f: f}
	for _, o := range opts {
		o(req)
	}
	// Initialize the filter connector.
	var err error
	if req.filters, err = joinfilters.New(name, req.clientOptions...); err != nil {
		return 0, errs.Wrapf(err, goredis.RetAddCronFail, "join filters NewFilters fail %v", err)
	}
	if req.trpcOptions, err = req.filters.LoadClientOptions(); err != nil {
		return 0, errs.Wrapf(err, goredis.RetAddCronFail, "filters. GetClientOptions fail %v", err)
	}
	var enable bool
	if enable, req.Spec, err = calcSpec(req.trpcOptions.Target); err != nil {
		return 0, err
	}
	// Do not start the timer.
	if !enable {
		return 0, nil
	}
	// Create a scheduled task.
	req.EntryID, err = c.cron.AddFunc(req.Spec, func() {
		c.callFunc(req)
	})
	if err != nil {
		return 0, errs.Wrapf(err, goredis.RetAddCronFail, "cron AddFunc fail %v", err)
	}
	return req.EntryID, nil
}

// Start starts the timer.
func (c *RedCron) Start() {
	c.cron.Start()
}

// Stop stops the timer.
func (c *RedCron) Stop() {
	c.cron.Stop()
}

// callFunc cron is a timer callback function.
func (c *RedCron) callFunc(req *Request) {
	// Generate context and add timeout.
	ctx := trpc.BackgroundContext()
	if req.trpcOptions.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.trpcOptions.Timeout)
		defer cancel()
	}
	entry := c.cron.Entry(req.EntryID)
	// Query whether the callback function needs to be executed.
	// Failure also needs to be reported via callback 007 monitoring report.
	if rsp := c.allow(ctx, req.filters.Name, req.Spec, &entry); rsp.IsRun || rsp.LastErr != nil {
		c.addFilters(ctx, req, rsp)
	}
}

// allow determines whether to allow execution, the return parameter must not be nil.
func (c *RedCron) allow(ctx context.Context, name, spec string, entry *cron.Entry) *Response {
	job := &pb.RedCronJob{}
	rsp := &Response{Now: time.Now(), Prev: entry.Prev, Next: entry.Next}
	var err error
	rsp.Cas, err = redcas.Get(ctx, c.cmdable, name).Unmarshal(job)
	if err != nil && err != redis.Nil {
		rsp.LastErr = err
		return rsp
	}
	rsp.Store = time.Unix(job.NextTime, 0)
	// If the spec is modified, allow immediate execution without interruption.
	if spec == job.Spec {
		// Indicates that other services have been executed and the state of the lock has been modified.
		if rsp.Now.Before(rsp.Store) {
			rsp.NotRunReason = "now before store"
			return rsp
		}
	}
	// Distributed lock grabbing operation through cas.
	job.NextTime = entry.Next.Unix()
	job.Spec = spec
	// Turn off the Redis alarm, abnormal error, cron filter will alarm.
	ctx, msg := goredis.WithMessage(ctx)
	msg.EnableFilter = false
	if _, err = redcas.Set(ctx, c.cmdable, name, job, rsp.Cas, 0).Result(); err != nil {
		// It means that the lock has been grabbed by other services.
		if errs.Code(err) == goredis.RetCASMismatch {
			rsp.NotRunReason = fmt.Sprintf("lock robbed %v", err)
			return rsp
		}
		rsp.LastErr = err
		return rsp
	}
	rsp.IsRun = true
	return rsp
}

// addFilters is integrated.
func (c *RedCron) addFilters(ctx context.Context, req *Request, rsp *Response) {
	// Modify 007 monitoring parameters.
	ctx, msg := codec.WithCloneMessage(ctx)
	defer codec.PutBackMessage(msg)
	msg.WithClientRPCName(fmt.Sprintf("/%s/cron", req.filters.Name))
	// Integrated filter, since the error has been reported to the filter,
	// such as: 007, log, so there is no need to deal with it here.
	_ = req.filters.Invoke(ctx, req, rsp,
		func(ctx context.Context, _ interface{}, _ interface{}) error {
			// Handle allow errors first.
			if rsp.LastErr != nil {
				return rsp.LastErr
			}
			return req.f(ctx, req, rsp)
		})
}

// Request is filter request body.
type Request struct {
	EntryID       cron.EntryID // Timing task ID.
	Spec          string       // cron timing time description string.
	f             filter.ClientHandleFunc
	clientOptions []client.Option
	trpcOptions   *client.Options
	filters       *joinfilters.Filters
}

// Response is the filter response body.
type Response struct {
	IsRun        bool      // Whether to be executed.
	NotRunReason string    // Reason for not running.
	Now          time.Time // Current time
	Store        time.Time // Redis stores time.
	Prev         time.Time // Current timer execution time.
	Next         time.Time // The next execution time of the timer.
	Cas          int64     // Redis CAS is convenient for locating problems.
	LastErr      error     // Last error.
}

// Option is parameter.
type Option func(r *Request)

// WithSpec modifies the timer time description.
func WithSpec(spec string) Option {
	return func(r *Request) {
		r.trpcOptions.Target = spec
	}
}

// WithTRPCOption adds trpc option。
func WithTRPCOption(trpcOptions ...client.Option) Option {
	return func(r *Request) {
		r.clientOptions = trpcOptions
	}
}

// calcSpec calculates cron spec.
func calcSpec(target string) (bool, string, error) {
	// The cron library does not support the syntax of not starting the timer, which is required by the business layer.
	// For example: the test environment does not need a timer, but the formal environment needs it,
	// and it can be closed by modifying the target to be empty.
	if target == "" {
		return false, "", nil
	}
	match := goredis.CronSelectorName + "://"
	if !strings.HasPrefix(target, match) {
		return false, "", errs.Newf(goredis.RetAddCronFail, "target %s format invalid, not %s start",
			target, match)
	}
	spec := target[len(match):]
	// Compatible with the target case.
	if spec == "" {
		return false, "", nil
	}
	return true, spec, nil
}
