package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginType = "database"
	pluginName = "kafka"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Config is kafka proxy configuration
type Config struct {
	MaxRequestSize  int32 `yaml:"max_request_size"`  // global maximum request body size
	MaxResponseSize int32 `yaml:"max_response_size"` // global maximum response body size
	RewriteLog      bool  `yaml:"rewrite_log"`       // whether to rewrite logs to log
}

// Plugin the default initialization of the plugin is used to load the kafka proxy connection parameter configuration
type Plugin struct{}

// Type plugin type
func (k *Plugin) Type() string {
	return pluginType
}

// Setup plugin initialization
func (k *Plugin) Setup(name string, configDesc plugin.Decoder) error {
	var config Config
	if err := configDesc.Decode(&config); err != nil {
		return err
	}
	if config.MaxRequestSize > 0 {
		sarama.MaxRequestSize = config.MaxRequestSize
	}
	if config.MaxResponseSize > 0 {
		sarama.MaxResponseSize = config.MaxResponseSize
	}
	if config.RewriteLog {
		sarama.Logger = LogReWriter{}
	}
	fmt.Println(config)

	return nil
}

// LogReWriter redirect log
type LogReWriter struct{}

// Print sarama.Logger interface
func (LogReWriter) Print(v ...interface{}) {
	log.Info(v...)
}

// Printf sarama.Logger interface
func (LogReWriter) Printf(format string, v ...interface{}) {
	log.Infof(format, v...)
}

// Println sarama.Logger interface
func (LogReWriter) Println(v ...interface{}) {
	log.Info(v...)
}
