package kafka

// config_parser.go parse configuration to Userconfig.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
)

// configParseFunc set UserConfig property.
type configParseFunc func(*UserConfig, string) error

// newConfigParsers returns the parameter entries that can be passed by the user.
// Users can only pass in defined parameter entries, and user content is assigned to userConfig through configParseFunc.
// Parameters have been classified according to their meanings and are included in different parse methods.
// newConfigParsers calls these methods and returns map[string]configParseFunc.
func newConfigParsers() map[string]configParseFunc {
	m := make(map[string]configParseFunc)
	parserBasicConfig(m)
	parserAdvanceConfig(m)
	parserBatchConfig(m)
	parserAuthConfig(m)
	parserDiscoverConfig(m)
	return m
}

// parserBasicConfig returns the necessary parameter items required to start Kafka.
// These parameter items can assign the user's input configuration to userConfig.
// The parameter items may validate the content of the user's input during execution,
// and actively throw exceptions or correct user input when encountering illegal content.
func parserBasicConfig(m map[string]configParseFunc) {
	m["clientid"] = func(config *UserConfig, val string) error {
		config.ClientID = val
		return nil
	}
	m["topics"] = func(config *UserConfig, val string) error {
		config.Topics = strings.Split(val, ",")
		return nil
	}
	m["topic"] = func(config *UserConfig, val string) error {
		config.Topic = val
		return nil
	}
	m["group"] = func(config *UserConfig, val string) error {
		config.Group = val
		return nil
	}
	m["async"] = func(config *UserConfig, val string) error {
		var err error
		config.Async, err = parseAsync(val)
		return err
	}
	m["compression"] = func(config *UserConfig, val string) error {
		var err error
		config.Compression, err = parseCompression(val)
		return err
	}
	m["version"] = func(config *UserConfig, val string) error {
		var err error
		config.Version, err = sarama.ParseKafkaVersion(val)
		return err
	}
	m["strategy"] = func(config *UserConfig, val string) error {
		var err error
		config.Strategy, err = parseStrategy(val)
		return err
	}
	m["partitioner"] = func(config *UserConfig, val string) error {
		var err error
		config.Partitioner, err = parsePartitioner(val)
		return err
	}
	m["fetchDefault"] = func(config *UserConfig, val string) error {
		var err error
		config.FetchDefault, err = strconv.Atoi(val)
		return err
	}
	m["fetchMax"] = func(config *UserConfig, val string) error {
		var err error
		config.FetchMax, err = strconv.Atoi(val)
		return err
	}
}

// parserAdvanceConfig advance config.
// These parameter items can assign the user's input configuration to userConfig.
// The parameter items may validate the content of the user's input during execution,
// and actively throw exceptions or correct user input when encountering illegal content.
func parserAdvanceConfig(m map[string]configParseFunc) {
	m["maxMessageBytes"] = func(config *UserConfig, val string) error {
		var err error
		config.MaxMessageBytes, err = strconv.Atoi(val)
		return err
	}
	m["flushMessages"] = func(config *UserConfig, val string) error {
		var err error
		config.FlushMessages, err = strconv.Atoi(val)
		return err
	}
	m["flushMaxMessages"] = func(config *UserConfig, val string) error {
		var err error
		config.FlushMaxMessages, err = strconv.Atoi(val)
		return err
	}
	m["flushBytes"] = func(config *UserConfig, val string) error {
		var err error
		config.FlushBytes, err = strconv.Atoi(val)
		return err
	}
	m["flushFrequency"] = func(config *UserConfig, val string) error {
		var err error
		config.FlushFrequency, err = parseDuration(val)
		return err
	}
	m["initial"] = func(config *UserConfig, val string) error {
		var err error
		config.Initial, err = parseInital(val)
		return err
	}
	m["maxWaitTime"] = func(config *UserConfig, val string) error {
		var err error
		config.MaxWaitTime, err = parseDuration(val)
		return err
	}
	m["batch"] = func(config *UserConfig, val string) error {
		var err error
		config.BatchConsumeCount, err = strconv.Atoi(val)
		return err
	}
	m["batchFlush"] = func(config *UserConfig, val string) error {
		var err error
		config.BatchFlush, err = parseDuration(val)
		return err
	}
	m["requiredAcks"] = func(config *UserConfig, val string) error {
		var err error
		config.RequiredAcks, err = parseRequireAcks(val)
		return err
	}
	m["maxRetry"] = func(config *UserConfig, val string) error {
		var err error
		config.MaxRetry, err = strconv.Atoi(val)
		config.ProducerRetry.Max = config.MaxRetry
		return err
	}
	m["trpcMeta"] = func(config *UserConfig, val string) error {
		var err error
		config.TrpcMeta, err = strconv.ParseBool(val)
		return err
	}
	m["idempotent"] = func(config *UserConfig, val string) error {
		var err error
		config.Idempotent, err = strconv.ParseBool(val)
		return err
	}
}

// parserBatchConfig returns the configParseFunc of batch consumption configuration items.
// These parameter items can assign the user's input configuration to userConfig.
// The parameter items may validate the content of the user's input during execution,
// and actively throw exceptions or correct user input when encountering illegal content.
func parserBatchConfig(m map[string]configParseFunc) {
	m["netMaxOpenRequests"] = func(config *UserConfig, val string) error {
		var err error
		config.NetMaxOpenRequests, err = strconv.Atoi(val)
		return err
	}
	m["maxProcessingTime"] = func(config *UserConfig, val string) error {
		var err error
		config.MaxProcessingTime, err = parseDuration(val)
		return err
	}
	m["netDailTimeout"] = func(config *UserConfig, val string) error {
		var err error
		config.NetDailTimeout, err = parseDuration(val)
		return err
	}
	m["netReadTimeout"] = func(config *UserConfig, val string) error {
		var err error
		config.NetReadTimeout, err = parseDuration(val)
		return err
	}
	m["netWriteTimeout"] = func(config *UserConfig, val string) error {
		var err error
		config.NetWriteTimeout, err = parseDuration(val)
		return err
	}
	m["groupSessionTimeout"] = func(config *UserConfig, val string) error {
		var err error
		config.GroupSessionTimeout, err = parseDuration(val)
		return err
	}
	m["groupRebalanceTimeout"] = func(config *UserConfig, val string) error {
		var err error
		config.GroupRebalanceTimeout, err = parseDuration(val)
		return err
	}
	m["groupRebalanceRetryMax"] = func(config *UserConfig, val string) error {
		var err error
		config.GroupRebalanceRetryMax, err = strconv.Atoi(val)
		return err
	}
	m["isolationLevel"] = func(config *UserConfig, val string) error {
		var err error
		config.IsolationLevel, err = parseIsolationLevel(val)
		return err
	}
	m["retryInterval"] = func(config *UserConfig, val string) error {
		var err error
		config.RetryInterval, err = parseRetryInterval(val)
		config.ProducerRetry.RetryInterval = config.RetryInterval
		return err
	}
}

// parserAuthConfig returns configParseFunc for kafka auth,
// These parameter items can assign the user's input configuration to userConfig.
// The parameter items may validate the content of the user's input during execution,
// and actively throw exceptions or correct user input when encountering illegal content.
func parserAuthConfig(m map[string]configParseFunc) {
	m["user"] = func(config *UserConfig, val string) error {
		var err error
		if config.ScramClient == nil {
			config.ScramClient = &LSCRAMClient{}
		}
		config.ScramClient.User = val
		return err
	}
	m["password"] = func(config *UserConfig, val string) error {
		var err error
		if config.ScramClient == nil {
			config.ScramClient = &LSCRAMClient{}
		}
		config.ScramClient.Password = val
		return err
	}
	m["mechanism"] = func(config *UserConfig, val string) error {
		var err error
		if config.ScramClient == nil {
			config.ScramClient = &LSCRAMClient{}
		}
		config.ScramClient.Mechanism = val
		return err
	}
	m["protocol"] = func(config *UserConfig, val string) error {
		var err error
		if config.ScramClient == nil {
			config.ScramClient = &LSCRAMClient{}
		}
		config.ScramClient.Protocol = val
		return err
	}
}

// parserDiscoverConfig returns configParseFunc for service discovery.
// These parameter items can assign the user's input configuration to userConfig.
// The parameter items may validate the content of the user's input during execution,
// and actively throw exceptions or correct user input when encountering illegal content.
func parserDiscoverConfig(m map[string]configParseFunc) {
	m["discover"] = func(config *UserConfig, discover string) error {
		config.discover = discover
		return nil
	}
	m["namespace"] = func(config *UserConfig, namespace string) error {
		config.namespace = namespace
		return nil
	}
	m["limiterRate"] = func(config *UserConfig, rate string) error {
		limiterRate, err := strconv.ParseFloat(rate, 64)
		if err != nil {
			return err
		}
		if limiterRate < 0 {
			return errors.New("param not support:limiterRate expect a value of no less than 0")
		}
		if config.RateLimitConfig == nil {
			config.RateLimitConfig = &RateLimitConfig{}
		}
		config.RateLimitConfig.Rate = limiterRate
		return nil
	}
	m["limiterBurst"] = func(config *UserConfig, burst string) error {
		limiterBurst, err := strconv.Atoi(burst)
		if err != nil {
			return err
		}
		if limiterBurst < 0 {
			return errors.New("param not support:limiterBurst expect a value of no less than 0")
		}
		if config.RateLimitConfig == nil {
			config.RateLimitConfig = &RateLimitConfig{}
		}
		config.RateLimitConfig.Burst = limiterBurst
		return nil
	}
}

// parseAsync async
func parseAsync(val string) (int, error) {
	if val == "1" {
		return 1, nil
	}
	return 0, nil
}

// parseCompression compression return different compression methods,
// please refer to sarama.CompressionCodec for details.
func parseCompression(val string) (sarama.CompressionCodec, error) {
	switch val {
	case "none":
		return sarama.CompressionNone, nil
	case "gzip":
		return sarama.CompressionGZIP, nil
	case "snappy":
		return sarama.CompressionSnappy, nil
	case "lz4":
		return sarama.CompressionLZ4, nil
	case "zstd":
		return sarama.CompressionZSTD, nil
	default:
		return sarama.CompressionNone, errors.New("param not support")
	}
}

// parseStrategy return different balance strategy,
// please refer to sarama.BalanceStrategy for details.
func parseStrategy(val string) (sarama.BalanceStrategy, error) {
	switch val {
	case "sticky":
		return sarama.BalanceStrategySticky, nil
	case "range":
		return sarama.BalanceStrategyRange, nil
	case "roundrobin":
		return sarama.BalanceStrategyRoundRobin, nil
	default:
		return nil, errors.New("param not support")
	}
}

// parsePartitioner return diffrent partitioner,
// please refer to sarama.Partitioner for details.
func parsePartitioner(val string) (func(topic string) sarama.Partitioner, error) {
	switch val {
	case "random":
		return sarama.NewRandomPartitioner, nil
	case "roundrobin":
		return sarama.NewRoundRobinPartitioner, nil
	case "hash":
		return sarama.NewHashPartitioner, nil
	default:
		return nil, errors.New("param not support")
	}
}

// parseInital
func parseInital(val string) (int64, error) {
	switch val {
	case "newest":
		return sarama.OffsetNewest, nil
	case "oldest":
		return sarama.OffsetOldest, nil
	default:
		return 0, errors.New("param not support")
	}
}

// parseDuration
func parseDuration(val string) (time.Duration, error) {
	maxWaitTime, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return time.Duration(maxWaitTime) * time.Millisecond, err
}

// parseRequireAcks
func parseRequireAcks(val string) (sarama.RequiredAcks, error) {
	ack, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	saramaAcks := sarama.RequiredAcks(ack)
	if saramaAcks != sarama.WaitForAll && saramaAcks != sarama.WaitForLocal && saramaAcks != sarama.NoResponse {
		return 0, fmt.Errorf("invalid requiredAcks: %s", val)
	}
	return saramaAcks, err
}

// parseIsolationLevel
func parseIsolationLevel(val string) (sarama.IsolationLevel, error) {
	switch val {
	case "ReadCommitted":
		return sarama.ReadCommitted, nil
	case "ReadUncommitted":
		return sarama.ReadUncommitted, nil
	default:
		return 0, errors.New("param not support")
	}
}

// parseRetryInterval
func parseRetryInterval(val string) (time.Duration, error) {
	timeInterval, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return time.Duration(timeInterval) * time.Millisecond, nil
}
