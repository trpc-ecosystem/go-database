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
	MaxSqlSize                int             `yaml:"max_sql_size"`
}

// Config is the struct for the configuration of the SQL proxy.
type Config struct {
	MaxIdle     int           `yaml:"max_idle"`     // Maximum number of idle connections.
	MaxOpen     int           `yaml:"max_open"`     // Maximum number of connections that can be open at same time.
	MaxLifetime int           `yaml:"max_lifetime"` // The maximum lifetime of each connection, in milliseconds.
	DriverName  string        `yaml:"driver_name"`  // Driver name for customization.
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
	if config.Logger != nil {
		loggers["*"] = NewTRPCLogger(logger.Config{
			SlowThreshold:             time.Duration(config.Logger.SlowThreshold) * time.Millisecond,
			Colorful:                  config.Logger.Colorful,
			IgnoreRecordNotFoundError: config.Logger.IgnoreRecordNotFoundError,
			LogLevel:                  config.Logger.LogLevel,
		})
		loggers["*"].maxSqlSize = config.Logger.MaxSqlSize
	}
	// Set pool configurations that effective for all GORM clients.
	defaultClientTransport.DefaultPoolConfig = PoolConfig{
		MaxIdle:     config.MaxIdle,
		MaxOpen:     config.MaxOpen,
		MaxLifetime: time.Duration(config.MaxLifetime) * time.Millisecond,
		DriverName:  config.DriverName,
	}
	setDefaultValueOfGlobalPoolConfig(&defaultClientTransport.DefaultPoolConfig)
	// Set pool configurations that effective for each GORM client.
	poolConfigs := make(map[string]PoolConfig, len(config.Service))
	for _, s := range config.Service {
		servicePoolConfig := PoolConfig{
			MaxIdle:     defaultClientTransport.DefaultPoolConfig.MaxIdle,
			MaxOpen:     defaultClientTransport.DefaultPoolConfig.MaxOpen,
			MaxLifetime: defaultClientTransport.DefaultPoolConfig.MaxLifetime,
			DriverName:  defaultClientTransport.DefaultPoolConfig.DriverName,
		}
		if s.MaxIdle != 0 {
			servicePoolConfig.MaxIdle = s.MaxIdle
		}
		if s.MaxOpen != 0 {
			servicePoolConfig.MaxOpen = s.MaxOpen
		}
		if s.MaxLifetime != 0 {
			servicePoolConfig.MaxLifetime = time.Duration(s.MaxLifetime) * time.Millisecond
		}
		if s.DriverName != "" {
			servicePoolConfig.DriverName = s.DriverName
		}
		poolConfigs[s.Name] = servicePoolConfig
		if s.Logger != nil {
			loggers[s.Name] = NewTRPCLogger(logger.Config{
				SlowThreshold:             time.Duration(s.Logger.SlowThreshold) * time.Millisecond,
				Colorful:                  s.Logger.Colorful,
				IgnoreRecordNotFoundError: s.Logger.IgnoreRecordNotFoundError,
				LogLevel:                  s.Logger.LogLevel,
			})
			loggers[s.Name].maxSqlSize = s.Logger.MaxSqlSize
		}
	}
	defaultClientTransport.PoolConfigs = poolConfigs
	// Need to call the register function explicitly, otherwise the configuration will not take effect.
	transport.RegisterClientTransport("gorm", defaultClientTransport)
	return nil
}

const (
	defaultMaxIdle     = 10
	defaultMaxOpen     = 10000
	defaultMaxLifetime = 3 * time.Minute
)

func setDefaultValueOfGlobalPoolConfig(pc *PoolConfig) {
	if pc.MaxIdle == 0 {
		pc.MaxIdle = defaultMaxIdle
	}
	if pc.MaxOpen == 0 {
		pc.MaxOpen = defaultMaxOpen
	}
	if pc.MaxLifetime == time.Duration(0) {
		pc.MaxLifetime = defaultMaxLifetime
	}
}
