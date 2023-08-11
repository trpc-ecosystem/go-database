package gorm

import (
	"time"

	"gorm.io/gorm/logger"

	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/transport"
)

const (
	pluginType = "database"
	pluginName = "gorm"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// loggers is a *TRPCLogger created from the logging configuration defined in the configuration file.
var loggers = map[string]*TRPCLogger{"*": DefaultTRPCLogger}

// getLogger returns a *TRPCLogger based on the loggers defined in the configuration file.
// If there is no logger defined in the configuration file, it returns the default logger(DefaultTRPCLogger).
func getLogger(name string) logger.Interface {
	trpcLogger, ok := loggers[name]
	if !ok {
		trpcLogger = loggers["*"]
	}
	return trpcLogger
}

// loggerConfig is the gorm logger configuration.
type loggerConfig struct {
	SlowThreshold             int64           `yaml:"slow_threshold"`
	Colorful                  bool            `yaml:"colorful"`
	IgnoreRecordNotFoundError bool            `yaml:"ignore_record_not_found_error"`
	LogLevel                  logger.LogLevel `yaml:"log_level"`
}

// Config is the struct for the configuration of the SQL proxy.
type Config struct {
	MaxIdle     int           `yaml:"max_idle"`     // Maximum number of idle connections.
	MaxOpen     int           `yaml:"max_open"`     // Maximum number of connections that can be open at same time.
	MaxLifetime int           `yaml:"max_lifetime"` // The maximum lifetime of each connection, in milliseconds.
	Logger      *loggerConfig `yaml:"logger"`       // Logger configuration.
	Service     []struct {
		// In the case of having multiple database connections,
		// you can configure the connection pool independently.
		Name        string
		MaxIdle     int `yaml:"max_idle"`     // Maximum number of idle connections.
		MaxOpen     int `yaml:"max_open"`     // Maximum number of connections that can be open at same time.
		MaxLifetime int `yaml:"max_lifetime"` // The maximum lifetime of each connection, in milliseconds.
		// The name of the custom driver used, which is empty by default.
		DriverName string        `yaml:"driver_name"`
		Logger     *loggerConfig `yaml:"logger"` // Logger configuration.
	}
}

// Plugin used to load the configuration of sql.DB connection parameters.
type Plugin struct{}

// Type returns the type of the plugin.
func (m *Plugin) Type() string {
	return pluginType
}

// Setup initializes the plugin.
// If the plugin has been specially configured in trpc_go.yaml,
// use the configuration to regenerate ClientTransport and register it into the tRPC framework
// to replace the default ClientTransport.
// This function will be executed in trpc.NewServer().
func (m *Plugin) Setup(name string, configDesc plugin.Decoder) (err error) {
	var config Config // YAML database connection configuration parameters.
	if err = configDesc.Decode(&config); err != nil {
		return
	}

	poolConfigs := make(map[string]PoolConfig, len(config.Service))
	if config.Logger != nil {
		loggers["*"] = NewTRPCLogger(logger.Config{
			SlowThreshold:             time.Duration(config.Logger.SlowThreshold) * time.Millisecond,
			Colorful:                  config.Logger.Colorful,
			IgnoreRecordNotFoundError: config.Logger.IgnoreRecordNotFoundError,
			LogLevel:                  config.Logger.LogLevel,
		})
	}
	for _, s := range config.Service {
		poolConfigs[s.Name] = PoolConfig{
			MaxIdle:     s.MaxIdle,
			MaxOpen:     s.MaxOpen,
			MaxLifetime: time.Duration(s.MaxLifetime) * time.Millisecond,
			DriverName:  s.DriverName,
		}
		if s.Logger != nil {
			loggers[s.Name] = NewTRPCLogger(logger.Config{
				SlowThreshold:             time.Duration(s.Logger.SlowThreshold) * time.Millisecond,
				Colorful:                  s.Logger.Colorful,
				IgnoreRecordNotFoundError: s.Logger.IgnoreRecordNotFoundError,
				LogLevel:                  s.Logger.LogLevel,
			})
		}
	}
	defaultClientTransport.PoolConfigs = poolConfigs

	defaultClientTransport.DefaultPoolConfig = PoolConfig{
		MaxIdle:     config.MaxIdle,
		MaxOpen:     config.MaxOpen,
		MaxLifetime: time.Duration(config.MaxLifetime) * time.Millisecond,
	}
	// Need to call the register function explicitly, otherwise the configuration will not take effect.
	transport.RegisterClientTransport("gorm", defaultClientTransport)
	return nil
}
