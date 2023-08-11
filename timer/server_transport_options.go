package timer

// ServerTransportOptions is server transport options.
type ServerTransportOptions struct {
	StartAtOnce bool // indicates whether to run the handle function when transport starts.
	Scheduler   Scheduler
}

// ServerTransportOption is function for server transport options
type ServerTransportOption func(*ServerTransportOptions)

// StartAtOnce sets transport to run the handle function at once when it starts.
func StartAtOnce() ServerTransportOption {
	return func(opts *ServerTransportOptions) {
		opts.StartAtOnce = true
	}
}

// WithScheduler sets the time scheduler for ServerTransport
func WithScheduler(s Scheduler) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.Scheduler = s
	}
}
