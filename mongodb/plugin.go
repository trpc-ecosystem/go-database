package mongodb

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/transport"
)

const (
	pluginType = "database"
	pluginName = "mongodb"
)

func init() {
	plugin.Register(pluginName, &mongoPlugin{})
}

// Config mongo is a proxy configuration structure declaration.
type Config struct {
	MinOpen        uint64        `yaml:"min_open"`        // Minimum number of simultaneous online connections
	MaxOpen        uint64        `yaml:"max_open"`        // The maximum number of simultaneous online connections
	MaxIdleTime    time.Duration `yaml:"max_idle_time"`   // Maximum idle time per link
	ReadPreference string        `yaml:"read_preference"` // reference on read
}

// mongoPlugin is used for plug-in default initialization,
// used to load mongo proxy connection parameter configuration.
type mongoPlugin struct{}

// Type is plugin type.
func (m *mongoPlugin) Type() string {
	return pluginType
}

// Setup is plugin initialization.
func (m *mongoPlugin) Setup(name string, configDesc plugin.Decoder) (err error) {
	var config Config // yaml database:mongo connection configuration parameters
	if err = configDesc.Decode(&config); err != nil {
		return
	}
	readMode, err := readpref.ModeFromString(config.ReadPreference)
	if err != nil {
		log.Errorf("readpref.ModeFromString failed, err=%v, set mod to primary", err)
		readMode = readpref.PrimaryMode
	}
	readF, err := readpref.New(readMode)
	if err != nil {
		log.Errorf("readpref.New failed, err=%v, set mod to primary", err)
		readF = readpref.Primary()
	}
	DefaultClientTransport = &ClientTransport{
		mongoDB:         make(map[string]*mongo.Client),
		MinOpenConns:    config.MinOpen,
		MaxOpenConns:    config.MaxOpen,
		MaxConnIdleTime: config.MaxIdleTime,
		ReadPreference:  readF,
		ServiceNameURIs: make(map[string][]string),
	}
	// You need to explicitly call register, otherwise the configuration will not take effect.
	transport.RegisterClientTransport(pluginName, DefaultClientTransport)
	return nil
}
