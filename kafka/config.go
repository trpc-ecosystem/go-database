package kafka

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
)

// DefaultClientID default client id
const DefaultClientID = "trpcgo"

// UserConfig configuration parsed from address
type UserConfig struct {
	Brokers         []string // cluster address
	Topics          []string // for consumers
	Topic           string   // for producers
	Group           string   // consumer group
	Async           int      // Whether to produce asynchronously, 0 is synchronous and 1 is asynchronous
	ClientID        string   // Client ID
	Compression     sarama.CompressionCodec
	Version         sarama.KafkaVersion
	Strategy        sarama.BalanceStrategy
	Partitioner     func(topic string) sarama.Partitioner
	Initial         int64 // The location where the new consumer group first connects to the cluster consumer
	FetchDefault    int
	FetchMax        int
	MaxWaitTime     time.Duration
	RequiredAcks    sarama.RequiredAcks
	ReturnSuccesses bool
	Timeout         time.Duration
	// When producing asynchronously
	MaxMessageBytes   int           // the maximum number of bytes in the local cache queue
	FlushMessages     int           // the maximum number of messages sent by the local cache broker
	FlushMaxMessages  int           // the maximum number of messages in the local cache queue
	FlushBytes        int           // the maximum number of bytes sent by the local cache broker
	FlushFrequency    time.Duration // In asynchronous production, the maximum time sent by the local cache broker
	BatchConsumeCount int           // Maximum number of messages for batch consumption
	BatchFlush        time.Duration // Batch consumption takes effect
	ScramClient       *LSCRAMClient // LSCRAM safety certification
	// The maximum number of retries on failure,
	// the default is 0: retry all the time <0 means no retry
	MaxRetry                       int
	NetMaxOpenRequests             int // Maximum number of requests
	MaxProcessingTime              time.Duration
	NetDailTimeout                 time.Duration
	NetReadTimeout                 time.Duration
	NetWriteTimeout                time.Duration
	GroupSessionTimeout            time.Duration
	GroupRebalanceTimeout          time.Duration
	GroupRebalanceRetryMax         int
	MetadataRetryMax               int
	MetadataRetryBackoff           time.Duration
	MetadataRefreshFrequency       time.Duration
	MetadataFull                   bool
	MetadataAllowAutoTopicCreation bool
	IsolationLevel                 sarama.IsolationLevel
	RetryInterval                  time.Duration // Retry Interval Works with MaxRetry
	ProducerRetry                  struct {
		Max           int           // Maximum number of retries
		RetryInterval time.Duration // RetryInterval retry interval
	}
	TrpcMeta   bool
	Idempotent bool // If enabled, the producer will ensure that exactly one copy of each message is written.

	discover        string           // The discovery type used for service discovery
	namespace       string           // service namespace
	RateLimitConfig *RateLimitConfig // token bucket limit configuration
}

// RateLimitConfig is limit config
type RateLimitConfig struct {
	Rate  float64 // token production rate
	Burst int     // token bucket capacity
}

func (uc *UserConfig) getServerConfig() *sarama.Config {
	sc := sarama.NewConfig()
	sc.Version = uc.Version
	if uc.ClientID == DefaultClientID {
		sc.ClientID = uc.Group
	} else {
		sc.ClientID = uc.ClientID
	}

	sc.Metadata.Retry.Max = uc.MetadataRetryMax
	sc.Metadata.Retry.Backoff = uc.MetadataRetryBackoff
	sc.Metadata.RefreshFrequency = uc.MetadataRefreshFrequency
	sc.Metadata.Full = uc.MetadataFull
	sc.Metadata.AllowAutoTopicCreation = uc.MetadataAllowAutoTopicCreation

	sc.Net.MaxOpenRequests = uc.NetMaxOpenRequests
	sc.Net.DialTimeout = uc.NetDailTimeout
	sc.Net.ReadTimeout = uc.NetReadTimeout
	sc.Net.WriteTimeout = uc.NetWriteTimeout

	sc.Consumer.MaxProcessingTime = uc.MaxProcessingTime
	sc.Consumer.Fetch.Default = int32(uc.FetchDefault)
	sc.Consumer.Fetch.Max = int32(uc.FetchMax)
	sc.Consumer.Offsets.Initial = uc.Initial
	sc.Consumer.Offsets.AutoCommit.Interval = 3 * time.Second // How often to submit consumption progress
	sc.Consumer.Group.Rebalance.Strategy = uc.Strategy
	sc.Consumer.Group.Rebalance.Timeout = uc.GroupRebalanceTimeout
	sc.Consumer.Group.Rebalance.Retry.Max = uc.GroupRebalanceRetryMax
	sc.Consumer.Group.Session.Timeout = uc.GroupSessionTimeout
	sc.Consumer.MaxWaitTime = uc.MaxWaitTime
	sc.Consumer.IsolationLevel = uc.IsolationLevel
	return sc
}

// GetDefaultConfig Get the default configuration
func GetDefaultConfig() *UserConfig {
	userConfig := &UserConfig{
		Brokers:     nil,
		Topics:      nil,
		Topic:       "",
		Group:       "",
		Async:       0,
		ClientID:    DefaultClientID,
		Compression: sarama.CompressionGZIP,       // CDMQ default compression
		Version:     sarama.V1_1_1_0,              // CDMQ default version
		Strategy:    sarama.BalanceStrategySticky, // CDMQ Strict by default
		Partitioner: sarama.NewRandomPartitioner,  // CDMQ default random
		Initial:     sarama.OffsetNewest,
		// FetchDefault the default number of message bytes to fetch.
		// For more details, see sarama.Config.Consumer.Fetch.Default
		FetchDefault: 524288,
		// FetchMax the maximum number of message bytes to fetch.
		// For more details, see sarama.Config.Consumer.Fetch.Max
		FetchMax: 1048576,
		// The maximum waiting time for a single consumption pull request.
		// The maximum wait time will only wait if there is no recent data.
		// This value should be set larger to reduce the consumption of empty requests on the QPS of the server.
		MaxWaitTime:                    time.Second,
		RequiredAcks:                   sarama.WaitForAll,
		ReturnSuccesses:                true,
		Timeout:                        time.Second, // Maximum request processing time on the server side
		MaxMessageBytes:                131072,      // CDMQ set up
		FlushMessages:                  0,
		FlushMaxMessages:               0,
		FlushBytes:                     0,
		FlushFrequency:                 0,
		BatchConsumeCount:              0,
		BatchFlush:                     2 * time.Second,
		ScramClient:                    nil,
		MaxRetry:                       0, // Unlimited retries, compatible with historical situations
		NetMaxOpenRequests:             5,
		MaxProcessingTime:              100 * time.Millisecond,
		NetDailTimeout:                 30 * time.Second,
		NetReadTimeout:                 30 * time.Second,
		NetWriteTimeout:                30 * time.Second,
		GroupSessionTimeout:            10 * time.Second,
		GroupRebalanceTimeout:          60 * time.Second,
		GroupRebalanceRetryMax:         4,
		MetadataRetryMax:               1,
		MetadataRetryBackoff:           1000 * time.Millisecond,
		MetadataRefreshFrequency:       600 * time.Second,
		MetadataFull:                   false, // disable pull all metadata
		MetadataAllowAutoTopicCreation: true,
		IsolationLevel:                 0,
		// Message consumption error retry interval The default is 3s The unit of this parameter is time.Millisecond
		RetryInterval: 3000 * time.Millisecond,
		// production retries the default configuration to align with the default configuration of sarama.NewConfig
		ProducerRetry: struct {
			Max           int
			RetryInterval time.Duration
		}{Max: 3, RetryInterval: 100 * time.Millisecond},
		TrpcMeta: false,
	}
	return userConfig
}

var (
	addrCfg     = make(map[string]*UserConfig)
	addrCfgLock sync.RWMutex
)

// RegisterAddrConfig register user-defined information, address is the corresponding address in the configuration file
func RegisterAddrConfig(address string, cfg *UserConfig) {
	addrCfgLock.Lock()
	defer addrCfgLock.Unlock()
	addrCfg[address] = cfg
}

// ParseAddress address format ip1:port1,ip2:port2?clientid=xx&topics=topic1,topic2&group=xxx&compression=gzip
func ParseAddress(address string) (*UserConfig, error) {
	addrCfgLock.RLock()
	cfg, ok := addrCfg[address]
	addrCfgLock.RUnlock() // The purpose of not using defer is to unlock as soon as possible and reduce the scope of locks
	if ok {
		return cfg, nil
	}

	config := GetDefaultConfig()
	tokens := strings.SplitN(address, "?", 2)
	if len(tokens) != 2 {
		return nil, fmt.Errorf("address format invalid: address: %v, tokens: %v", address, tokens)
	}

	config.Brokers = strings.Split(tokens[0], ",")
	values, err := url.ParseQuery(tokens[1])
	if err != nil {
		return nil, fmt.Errorf("address format invalid: brokers: %v, params: %v, err: %w",
			config.Brokers, tokens[1], err)
	}

	configParsers := newConfigParsers()
	for key := range values {
		value := values.Get(key)
		if value == "" {
			return nil, fmt.Errorf("address format invalid, key: %v, value is empty", key)
		}

		// find parser by  key
		f := configParsers[key]
		if f == nil {
			return nil, fmt.Errorf("address format invalid, unknown key: %v", key)
		}

		// apply value to config
		if err = f(config, value); err != nil {
			return nil, fmt.Errorf("address format invalid, key: %v, value: %v, err:%w", key, value, err)
		}
	}
	// Service discovery node analysis
	if config.discover == "" {
		return config, nil
	}
	if config.namespace == "" {
		return nil, fmt.Errorf("namespace format invalid:namespace cannot be empty when discover is not empty")
	}
	config.Brokers, err = getBrokers(config.Brokers, config.discover, config.namespace)
	if err != nil {
		return nil, fmt.Errorf("fail to get brokers, discover:%s, namespace:%s, err:%w",
			config.discover, config.namespace, err)
	}
	return config, nil
}

// getBrokers get the service node in the configuration
func getBrokers(brokers []string, discoverName, namespace string) ([]string, error) {
	if len(brokers) == 0 {
		return brokers, nil
	}
	discover := discovery.Get(discoverName)
	if discover == nil {
		return nil, fmt.Errorf("fail to get discovery: %s discovery is not register", discoverName)
	}
	nodes, err := discover.List(brokers[0], discovery.WithNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("fail to get %s nodes,service:%s,err:%w", discoverName, brokers[0], err)
	}
	router := servicerouter.Get(discoverName)
	if router == nil {
		return nil, fmt.Errorf("fail to get servicerouter: %s servicerouter is not register", discoverName)
	}
	filteredNodes, err := router.Filter(brokers[0], nodes)
	if err != nil {
		return nil, fmt.Errorf("fail to get service router,service:%s, err: %w", brokers[0], err)
	}
	var services []string
	for _, node := range filteredNodes {
		services = append(services, node.Address)
	}
	return services, nil
}
