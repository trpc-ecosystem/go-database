package goes

import "trpc.group/trpc-go/trpc-go/plugin"

const (
	pluginType = "database" // plugin type
	pluginName = "goes"     // plugin name
)

func init() {
	plugin.Register(pluginName, &Plugin{}) // register plugin and obtain services config
}

// LogConfig is log configuration
type LogConfig struct {
	Enabled         bool `yaml:"enabled"`          // enable log
	RequestEnabled  bool `yaml:"request_enabled"`  // enable request body
	ResponseEnabled bool `yaml:"response_enabled"` // enable response body
}

// ClientOption  goes database connection option
type ClientOption struct {
	Name     string    `yaml:"name"`     // name
	URL      string    `yaml:"url"`      // url
	User     string    `yaml:"user"`     // user
	Password string    `yaml:"password"` // password
	Timeout  int       `yaml:"timeout"`  // timeout
	Log      LogConfig `yaml:"log"`      // log
}

// Config goes config struct
type Config struct {
	ClientsOptions []*ClientOption `yaml:"clientoptions"` // goes database option slice
}

// Plugin plugin struct
type Plugin struct{}

// Type plugin.Factory interface implementation
func (m *Plugin) Type() string {
	return pluginType
}

// goes database config map, key:"service name", value:*config
var (
	dbConfigMap = make(map[string]*ClientOption)
)

// goes database connection default timeout
const defaultTimeoutValue = 1000

// set default options values
func (o *ClientOption) setDefault() {
	// set connection default timeout
	if o.Timeout == 0 {
		o.Timeout = defaultTimeoutValue
	}
}

// Setup plugin.Factory interface implementation
func (m *Plugin) Setup(name string, configDesc plugin.Decoder) error {
	var config Config // yaml database goes connection options
	if err := configDesc.Decode(&config); err != nil {
		return err
	}

	// init db config map
	for _, clientOption := range config.ClientsOptions {
		dbConfigMap[clientOption.Name] = clientOption
		clientOption.setDefault()
	}
	return nil
}
