package kafka

import (
	"fmt"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
)

// TestParseAddress test parsing address
func TestParseAddress(t *testing.T) {
	cfg, err := ParseAddress("test1.kafka.com:1234,test2.kafka.com:5678?clientid=xxxx&topic=abc&fetchDefault=12345678&fetchMax=54455&maxMessageBytes=35326236&initial=oldest" +
		"&flushMessages=10&flushMaxMessages=100&flushBytes=10000000&flushFrequency=100&idempotent=true")
	assert.Nil(t, err)
	assert.Equal(t, []string{"test1.kafka.com:1234", "test2.kafka.com:5678"}, cfg.Brokers)
	assert.Equal(t, "xxxx", cfg.ClientID)
	assert.Equal(t, "abc", cfg.Topic)
	assert.Equal(t, 12345678, cfg.FetchDefault)
	assert.Equal(t, 54455, cfg.FetchMax)
	assert.Equal(t, 35326236, cfg.MaxMessageBytes)
	assert.Equal(t, sarama.OffsetOldest, cfg.Initial)
	assert.Equal(t, 10, cfg.FlushMessages)
	assert.Equal(t, 100, cfg.FlushMaxMessages)
	assert.Equal(t, 10000000, cfg.FlushBytes)
	assert.Equal(t, 100*time.Millisecond, cfg.FlushFrequency)
	assert.Equal(t, true, cfg.Idempotent)

	RegisterAddrConfig("test_registered", cfg)

	cfgForCache, err := ParseAddress("test_registered")
	assert.Nil(t, err)
	assert.Equal(t, cfg, cfgForCache)
}

// Test_parseAddress test parsing address function
func Test_parseAddress(t *testing.T) {
	address := "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&user=kafka_test&password=cccaaabb&mechanism=SCRAM-SHA-512"
	_, err := ParseAddress(address)
	assert.Nil(t, err)
	address = "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&protocol=SASL_SSL&user=kafka_test&password=cccaaabb&mechanism=SCRAM-SHA-512"
	_, err = ParseAddress(address)
	assert.Nil(t, err)
	address = "127.0.0.1:9092?topics=Topic1,Topic2,Topic3&clientid=client1&version=0.10.2.0&strategy=sticky&batch=2&batchFlush=3000&group=test&maxRetry=10"
	_, err = ParseAddress(address)
	assert.Nil(t, err)

	address = ""
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
	address = "127.0.0.1:9092?topics=Topic1,Topic2,Topic3&test"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
	address = "127.0.0.1:9092?test=test"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
	address = "127.0.0.1:9092?%A="
	_, err = ParseAddress(address) // It returns an error if any % is not followed by two hexadecimal digits.
	assert.NotNil(t, err)
	address = "trpc.name.kafka.server?discover=polaris&maxWaitTime=250"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
	address = "trpc.name.kafka.server?discover=polaris&maxWaitTime=250&namespace=Development"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
	address = "trpc.name.kafka.server?limiterRate=10&limiterBurst=20"
	_, err = ParseAddress(address)
	assert.Nil(t, err)
}

// Test_parseAddressWithmaxWaitTime test resolve address function - with max wait time
func Test_parseAddressWithmaxWaitTime(t *testing.T) {
	address := "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&maxWaitTime=250"
	conf, err := ParseAddress(address)
	assert.Nil(t, err)
	assert.Equal(t, 250*time.Millisecond, conf.MaxWaitTime)
	address = "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&maxWaitTime=test"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
}

func Test_parseAddressWithMetadata(t *testing.T) {
	address := "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&metadataRetryMax=3&metadataRetryBackoff=10&metadataRefreshFrequency=100&metadataFull=true&metadataAllowAutoTopicCreation=true"
	conf, err := ParseAddress(address)
	assert.Nil(t, err)
	assert.Equal(t, 3, conf.MetadataRetryMax)
	assert.Equal(t, 10*time.Millisecond, conf.MetadataRetryBackoff)
	assert.Equal(t, 100*time.Second, conf.MetadataRefreshFrequency)
	assert.Equal(t, true, conf.MetadataFull)
	assert.Equal(t, true, conf.MetadataAllowAutoTopicCreation)
	address = "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&metadataRetryMax=-3&metadataRetryBackoff=10&metadataRefreshFrequency=100"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
	address = "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&metadataRetryMax=3&metadataRetryBackoff=-10&metadataRefreshFrequency=100"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
	address = "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&metadataRetryMax=3&metadataRetryBackoff=10&metadataRefreshFrequency=-100"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
}

// Test_parseAddressWithmaxWaitoutTime test resolve address function - no maximum wait time
func Test_parseAddressWithmaxWaitoutTime(t *testing.T) {
	address := "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&user=kafka_test&password=cccaaabb&mechanism=SCRAM-SHA-512"
	conf, err := ParseAddress(address)
	assert.Nil(t, err)
	assert.Equal(t, 1000*time.Millisecond, conf.MaxWaitTime)
}

// Test_parseAddressWithRequireAcks test parsing address function -requiredAcks
func Test_parseAddressWithRequiredAcks(t *testing.T) {
	address := "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&user=kafka_test&password=cccaaabb&mechanism=SCRAM-SHA-512&requiredAcks=1&retryInterval=2000"
	conf, err := ParseAddress(address)
	assert.Nil(t, err)
	assert.Equal(t, sarama.WaitForLocal, conf.RequiredAcks)
}

// Test_parserDiscoverConfig test
func Test_parserDiscoverConfig(t *testing.T) {
	m := newConfigParsers()
	parserDiscoverConfig(m)
}

func Test_parseAddressWithNetTimeout(t *testing.T) {
	address := "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&netMaxOpenRequests=1000&maxProcessingTime=1000&netDailTimeout=1000&netReadTimeout=1000&netWriteTimeout=1000&groupSessionTimeout=1000&groupRebalanceTimeout=1000&groupRebalanceRetryMax=3"
	_, err := ParseAddress(address)
	assert.Nil(t, err)
}

func Test_parseAddressWithIsolationLevel(t *testing.T) {
	address := "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&netMaxOpenRequests=1000&maxProcessingTime=1000&netDailTimeout=1000&netReadTimeout=1000&netWriteTimeout=1000&groupSessionTimeout=1000&groupRebalanceTimeout=1000&groupRebalanceRetryMax=3&isolationLevel=ReadCommitted"
	_, err := ParseAddress(address)
	assert.Nil(t, err)
	address = "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&netMaxOpenRequests=1000&maxProcessingTime=1000&netDailTimeout=1000&netReadTimeout=1000&netWriteTimeout=1000&groupSessionTimeout=1000&groupRebalanceTimeout=1000&groupRebalanceRetryMax=3&isolationLevel=ReadUncommitted"
	_, err = ParseAddress(address)
	assert.Nil(t, err)
	address = "127.0.0.1:9092?topic=Topic1&clientid=client1&compression=none&partitioner=hash&netMaxOpenRequests=1000&maxProcessingTime=1000&netDailTimeout=1000&netReadTimeout=1000&netWriteTimeout=1000&groupSessionTimeout=1000&groupRebalanceTimeout=1000&groupRebalanceRetryMax=3&isolationLevel=readCommite"
	_, err = ParseAddress(address)
	assert.NotNil(t, err)
}

func Test_GetDefaultConfig(t *testing.T) {
	_ = GetDefaultConfig()
	assert.Nil(t, nil)
}

func Test_parseAsync(t *testing.T) {
	_, err := parseAsync("1")
	assert.Nil(t, err)
	_, err = parseAsync("0")
	assert.Nil(t, err)
}

func Test_parseCompression(t *testing.T) {
	strs := []string{"none", "gzip", "snappy", "lz4", "zstd", "default"}
	for _, item := range strs {
		_, _ = parseCompression(item)
	}
	assert.Nil(t, nil)
}

func Test_parseStrategy(t *testing.T) {
	strs := []string{"none", "sticky", "range", "roundrobin"}
	for _, item := range strs {
		_, _ = parseStrategy(item)
	}
	assert.Nil(t, nil)
}

func Test_parsePartitioner(t *testing.T) {
	strs := []string{"none", "random", "hash", "roundrobin"}
	for _, item := range strs {
		_, _ = parsePartitioner(item)
	}
	assert.Nil(t, nil)
}

func Test_parseInital(t *testing.T) {
	strs := []string{"none", "newest", "oldest"}
	for _, item := range strs {
		_, _ = parseInital(item)
	}
	assert.Nil(t, nil)
}

func Test_parseDuration(t *testing.T) {
	_, err := parseDuration("1000")
	assert.Nil(t, err)
}

func Test_parseRequireAcks(t *testing.T) {
	acks, _ := parseRequireAcks("1")
	assert.Equal(t, acks, sarama.WaitForLocal)
	acks, _ = parseRequireAcks("-1")
	assert.Equal(t, acks, sarama.WaitForAll)
	acks, _ = parseRequireAcks("0")
	assert.Equal(t, acks, sarama.NoResponse)
	acks, _ = parseRequireAcks("test")
	assert.Equal(t, acks, sarama.NoResponse)
	acks, _ = parseRequireAcks("100")
	assert.Equal(t, acks, sarama.NoResponse)
}

func Test_parseRetryInterval(t *testing.T) {
	interval, err := parseRetryInterval("wrong case")
	assert.Equal(t, time.Duration(0), interval)
	assert.NotNil(t, err)

	interval, err = parseRetryInterval("1000")
	assert.Equal(t, 1000*time.Millisecond, interval)
	assert.Nil(t, err)
}

func Test_getServerConfig_ClientID(t *testing.T) {
	userConfig := GetDefaultConfig()
	userConfig.Group = "test_group"
	kafkaClientConfig := userConfig.getServerConfig()
	assert.Equal(t, kafkaClientConfig.ClientID, userConfig.Group)

	userConfig.ClientID = "test_client"
	kafkaClientConfig = userConfig.getServerConfig()
	assert.Equal(t, kafkaClientConfig.ClientID, userConfig.ClientID)
}

// Test_getBrokers get polaris ip test
func Test_getBrokers(t *testing.T) {
	t.Run("getBorkers discovery not register", func(t *testing.T) {
		// register mock discovery and servicerouter
		_, err := getBrokers([]string{"trpc.***.kafka.test"}, "polaris", "Development")
		assert.Equal(t, err.Error(), "fail to get discovery: polaris discovery is not register")
	})
	t.Run("getBorkers service_router not register", func(t *testing.T) {
		// register mock discovery and servicerouter
		discovery.Register("polaris", &mock_discovery{})
		_, err := getBrokers([]string{"trpc.***.kafka.test"}, "polaris", "Development")
		assert.Equal(t, err.Error(), "fail to get servicerouter: polaris servicerouter is not register")
	})
	t.Run("getBorkers call discover.List error", func(t *testing.T) {
		// register mock discovery and servicerouter
		discovery.Register("polaris", &mock_discovery{})
		servicerouter.Register("polaris", &mock_service_router{})
		_, err := getBrokers([]string{"trpc.****.kafka.test"}, "polaris", "Development")
		assert.NotNil(t, err)
	})
	t.Run("getBorkers", func(t *testing.T) {
		// register mock discovery and servicerouter
		discovery.Register("polaris", &mock_discovery{})
		servicerouter.Register("polaris", &mock_service_router{})
		got, err := getBrokers([]string{"trpc.***.kafka.test"}, "polaris", "Development")
		assert.Nil(t, err)
		assert.Equal(t, got, []string{"127.0.0.1:0"})
	})
}

type mock_discovery struct{}

// List implement discovery.List
func (d *mock_discovery) List(serviceName string, opt ...discovery.Option) ([]*registry.Node, error) {
	if serviceName != "trpc.***.kafka.test" {
		return nil, fmt.Errorf("service is not found:%s", serviceName)
	}
	return []*registry.Node{
		{
			ServiceName: "mock_discovery_service",
			Address:     "127.0.0.1:0",
		},
	}, nil
}

type mock_service_router struct{}

// Filter implement servicerouter.Filter
func (d *mock_service_router) Filter(serviceName string, nodes []*registry.Node, opt ...servicerouter.Option) ([]*registry.Node, error) {
	return []*registry.Node{
		{
			ServiceName: "mock_discovery_service",
			Address:     "127.0.0.1:0",
		},
	}, nil
}
