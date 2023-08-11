package mysql

import (
	"database/sql"
	"time"

	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/transport"
)

const (
	pluginType = "database"
	pluginName = "mysql"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Config mysql Proxy configuration structure declaration.
type Config struct {
	MaxIdle     int    `yaml:"max_idle"`     // Maximum number of idle connections
	MaxOpen     int    `yaml:"max_open"`     // Maximum number of simultaneous online connections
	MaxLifetime int    `yaml:"max_lifetime"` // Maximum lifetime per connection, in milliseconds
	DriverName  string `yaml:"driver_name"`  // The driver name used, default is mysql
}

func (c *Config) setDefault() {
	if c.DriverName == "" {
		c.DriverName = defaultDriverName
	}
}

// Plugin proxy plugin is initialized by default, and is used to load mysql proxy connection parameters.
type Plugin struct{}

// Type implements the plugin.
func (m *Plugin) Type() string {
	return pluginType
}

// Setup implements the plugin.Factory interface.
func (m *Plugin) Setup(name string, configDesc plugin.Decoder) (err error) {
	var config Config // yaml database:mysql connection configuration parameters.
	if err = configDesc.Decode(&config); err != nil {
		return
	}
	config.setDefault()
	DefaultClientTransport = &ClientTransport{
		dbs:         make(map[string]*sql.DB),
		MaxIdle:     config.MaxIdle,
		MaxOpen:     config.MaxOpen,
		MaxLifetime: time.Duration(config.MaxLifetime) * time.Millisecond,
		DriverName:  config.DriverName,
	}
	// register needs to be called explicitly, otherwise the configuration will not take effect.
	transport.RegisterClientTransport("mysql", DefaultClientTransport)
	return nil
}
