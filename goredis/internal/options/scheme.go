package options

import (
	"fmt"
	"strings"

	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"trpc.group/trpc-go/trpc-go/errs"
)

// parseScheme parses address.
func parseScheme(scheme, namespace, host string, isProxy bool) ([]string, error) {
	var addrs []string
	switch scheme {
	case "ip", RedisSelectorName, RedissSelectorName:
		addrs = parseRedisScheme(host)
	case "polaris":
		if isProxy {
			addrs = parsePolarisProxyScheme()
			break
		}
		var err error
		if addrs, err = parsePolarisClusterScheme(namespace, host); err != nil {
			return nil, err
		}
	default:
		return nil, errs.Newf(RetParseConfigFail, "scheme %s not support", scheme)
	}
	if len(addrs) == 0 {
		return nil, errs.Newf(RetParseConfigFail, "parseScheme addrs empty")
	}
	return addrs, nil
}

// parseIPScheme is parsing IP patterns.
func parseRedisScheme(host string) []string {
	rawAddrs := strings.Split(host, ",")
	addrs := make([]string, 0, len(rawAddrs))
	for _, rawAddr := range rawAddrs {
		addr := strings.TrimSpace(rawAddr)
		if addr != "" {
			addrs = append(addrs, addr)
		}
	}
	return addrs
}

// parsePolarisProxyScheme is parsing Polaris Proxy Mode.
func parsePolarisProxyScheme() []string {
	// Just fill in an address, and modify it later through DialHook.
	// See：https://github.com/redis/go-redis/blob/v9.0.2/options.go#L138
	return []string{"localhost:6379"}
}

// parsePolarisClusterScheme is parsing Polaris cluster mode.
func parsePolarisClusterScheme(namespace, service string) ([]string, error) {
	consumer, err := api.NewConsumerAPI()
	if err != nil {
		return nil, errs.Wrapf(err, RetParseConfigFail, "NewConsumerAPI fail %v", err)
	}
	defer consumer.Destroy()
	req := &api.GetAllInstancesRequest{
		GetAllInstancesRequest: model.GetAllInstancesRequest{
			Service:   service,
			Namespace: namespace,
		},
	}
	rsp, err := consumer.GetAllInstances(req)
	if nil != err {
		return nil, errs.Wrapf(err, RetParseConfigFail, "parsePolarisTarget GetInstances %v", err)
	}
	addrs := make([]string, 0, len(rsp.Instances))
	for _, instance := range rsp.Instances {
		addrs = append(addrs, fmt.Sprintf("%s:%d", instance.GetHost(), instance.GetPort()))
	}
	return addrs, nil
}
