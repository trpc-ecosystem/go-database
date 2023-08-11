// Package joinfilters is a trpc filter adapter.
package joinfilters

import (
	"context"
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
	"trpc.group/trpc-go/trpc-go/transport"
)

// Filters is filter adapter.
type Filters struct {
	Name string
	Opts []client.Option
}

// New creates a collection of filters.
func New(name string, opts ...client.Option) (*Filters, error) {
	f := &Filters{
		Name: name,
		Opts: opts,
	}
	return f, nil
}

// Invoke callbacks mode request.
func (f *Filters) Invoke(ctx context.Context, req, rsp interface{}, call filter.ClientHandleFunc,
	opts ...client.Option) error {
	// Get filter.
	options, err := f.LoadClientOptions(opts...)
	if err != nil {
		return err
	}
	// Set timeout.
	if options.Timeout > 0 {
		var cancel context.CancelFunc // Calculation timed out.
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}
	// Execute filter.
	return options.Filters.Filter(ctx, req, rsp, call)
}

// LoadClientOptions parses the caller service configuration.
func (f *Filters) LoadClientOptions(opts ...client.Option) (*client.Options, error) {
	// Parsing configuration parameters
	clientOptions := &client.Options{
		Transport:                transport.DefaultClientTransport,
		Selector:                 selector.DefaultSelector,
		SerializationType:        -1, // Initial value -1, do not set
		CurrentSerializationType: -1, // The serialization method of the current client.
		// The serialization method in the protocol is based on the SerializationType,
		// and the forwarding proxy situation.
		CurrentCompressType: -1, // The current client transparent transmission body is not serialized,
		// but the backend of the business agreement needs to specify the serialization method
	}
	// Use the servicename (package.service) of the protocol file
	// of the transferred party as the key to obtain the relevant configuration.
	err := loadClientConfig(clientOptions, f.Name)
	if err != nil {
		return nil, err
	}
	// The input parameter is the highest priority to overwrite the original data.
	for _, o := range f.Opts {
		o(clientOptions)
	}
	for _, o := range opts {
		o(clientOptions)
	}
	err = loadClientFilterConfig(clientOptions, f.Name)
	if err != nil {
		return nil, err
	}
	return clientOptions, nil
}

func loadClientConfig(opts *client.Options, key string) error {
	cfg, ok := client.DefaultClientConfig()[key]
	if !ok {
		return nil
	}
	if err := setNamingOptions(opts, cfg); err != nil {
		return err
	}

	if cfg.Timeout > 0 {
		opts.Timeout = time.Duration(cfg.Timeout) * time.Millisecond
	}
	if cfg.Serialization != nil {
		opts.SerializationType = *cfg.Serialization
	}

	if cfg.Compression > codec.CompressTypeNoop {
		opts.CompressType = cfg.Compression
	}
	if cfg.Protocol != "" {
		o := client.WithProtocol(cfg.Protocol)
		o(opts)
	}
	if cfg.Network != "" {
		opts.Network = cfg.Network
		opts.CallOptions = append(opts.CallOptions, transport.WithDialNetwork(cfg.Network))
	}
	if cfg.Password != "" {
		opts.CallOptions = append(opts.CallOptions, transport.WithDialPassword(cfg.Password))
	}
	if cfg.CACert != "" {
		opts.CallOptions = append(opts.CallOptions,
			transport.WithDialTLS(cfg.TLSCert, cfg.TLSKey, cfg.CACert, cfg.TLSServerName))
	}
	return nil
}

func setNamingOptions(opts *client.Options, cfg *client.BackendConfig) error {
	if cfg.ServiceName != "" {
		opts.ServiceName = cfg.ServiceName
	}
	if cfg.Namespace != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithNamespace(cfg.Namespace))
	}
	if cfg.EnvName != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDestinationEnvName(cfg.EnvName))
	}
	if cfg.SetName != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDestinationSetName(cfg.SetName))
	}
	if cfg.DisableServiceRouter {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDisableServiceRouter())
		opts.DisableServiceRouter = true
	}
	if cfg.Target != "" {
		opts.Target = cfg.Target
		return nil
	}
	if cfg.Discovery != "" {
		d := discovery.Get(cfg.Discovery)
		if d == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: discovery %s no registered", cfg.Discovery))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDiscovery(d))
	}
	if cfg.ServiceRouter != "" {
		r := servicerouter.Get(cfg.ServiceRouter)
		if r == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: servicerouter %s no registered", cfg.ServiceRouter))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithServiceRouter(r))
	}
	if cfg.Loadbalance != "" {
		balancer := loadbalance.Get(cfg.Loadbalance)
		if balancer == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: balancer %s no registered", cfg.Loadbalance))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithLoadBalancer(balancer))
	}
	if cfg.Circuitbreaker != "" {
		cb := circuitbreaker.Get(cfg.Circuitbreaker)
		if cb == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: circuitbreaker %s no registered", cfg.Circuitbreaker))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithCircuitBreaker(cb))
	}
	return nil
}

func loadClientFilterConfig(opts *client.Options, key string) error {
	if opts.DisableFilter {
		opts.Filters = filter.EmptyChain
		return nil
	}
	cfg, ok := client.DefaultClientConfig()[key]
	if !ok {
		return nil
	}
	for _, filterName := range cfg.Filter {
		f := filter.GetClient(filterName)
		if f == nil {
			return fmt.Errorf("client config: filter %s no registered", filterName)
		}
		opts.Filters = append(opts.Filters, f)
		opts.FilterNames = append(opts.FilterNames, filterName)
	}
	return nil
}
