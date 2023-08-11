package gorm

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm/logger"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/transport"

	. "github.com/smartystreets/goconvey/convey"
)

// TestUnit_GormPlugin_Type_P0 GormPlugin.Type test case.
func TestUnit_GormPlugin_Type_P0(t *testing.T) {
	Convey("TestUnit_GormPlugin_Type_P0", t, func() {
		sqlPlugin := new(Plugin)
		So(sqlPlugin.Type(), ShouldEqual, pluginType)
	})
}

// TestUnit_GormPlugin_Setup_P0 GormPlugin.Setup test case.
func TestUnit_GormPlugin_Setup_P0(t *testing.T) {
	Convey("TestUnit_GormPlugin_Setup_P0", t, func() {
		mp := &Plugin{}
		Convey("Config Decode Fail", func() {
			err := mp.Setup(pluginName, &plugin.YamlNodeDecoder{Node: nil})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "yaml node empty")
		})
		Convey("Setup Success", func() {
			var bts = `
plugins:
  database:
    gorm:
      max_idle: 20
      max_open: 100
      max_lifetime: 180
      logger:
        slow_threshold: 1000
        colorful: true
        ignore_record_not_found_error: true
        log_level: 4
      service:
        - name: trpc.mysql.test.db1
          max_idle: 10
          max_open: 20
          max_lifetime: 10000
          logger:
            slow_threshold: 1000
            colorful: true
            ignore_record_not_found_error: true
            log_level: 4
        - name: trpc.mysql.test.db2
          max_idle: 5
          max_open: 10
          max_lifetime: 20000
          logger:
            slow_threshold: 3000
            colorful: false
            ignore_record_not_found_error: false
            log_level: 3
`

			var cfg = trpc.Config{}
			err := yaml.Unmarshal([]byte(bts), &cfg)
			assert.Nil(t, err)
			var yamlNode *yaml.Node
			if configP, ok := cfg.Plugins[pluginType]; ok {
				if node, ok := configP[pluginName]; ok {
					yamlNode = &node
				}
			}
			So(yamlNode, ShouldNotBeNil)

			err = mp.Setup(pluginName, &plugin.YamlNodeDecoder{Node: yamlNode})
			So(err, ShouldBeNil)

			ct := transport.GetClientTransport(pluginName)
			clientTransport, ok := ct.(*ClientTransport)
			So(ok, ShouldBeTrue)

			expectedTransport := &ClientTransport{
				opts:  &transport.ClientTransportOptions{},
				SQLDB: make(map[string]*sql.DB),
				DefaultPoolConfig: PoolConfig{
					MaxIdle:     20,
					MaxOpen:     100,
					MaxLifetime: 180 * time.Millisecond,
				},
				PoolConfigs: map[string]PoolConfig{
					"trpc.mysql.test.db1": {
						MaxIdle:     10,
						MaxOpen:     20,
						MaxLifetime: 10000 * time.Millisecond,
					},
					"trpc.mysql.test.db2": {
						MaxIdle:     5,
						MaxOpen:     10,
						MaxLifetime: 20000 * time.Millisecond,
					},
				},
			}
			// cmp.opener is of type func which is not comparable, so only compare the unexported fields if needed.
			So(cmp.Equal(clientTransport.opts, expectedTransport.opts), ShouldBeTrue)
			So(cmp.Equal(clientTransport, expectedTransport, cmpopts.IgnoreUnexported(ClientTransport{}),
				cmpopts.IgnoreFields(ClientTransport{}, "SQLDBLock")), ShouldBeTrue)
		})
	})
}

func TestUnit_getLogger(t *testing.T) {
	Convey("TestUnit_getLogger", t, func() {
		loggers = map[string]*TRPCLogger{"*": DefaultTRPCLogger}
		trpcLogger := getLogger("trpc.mysql.defaultLogger.db")
		So(trpcLogger, ShouldEqual, DefaultTRPCLogger)
		loggers["trpc.mysql.customLogger.db"] = NewTRPCLogger(logger.Config{
			SlowThreshold:             1000 * time.Millisecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: true,
			LogLevel:                  3,
		})
		trpcLogger = getLogger("trpc.mysql.customLogger.db")
		So(trpcLogger, ShouldNotBeNil)
		So(trpcLogger, ShouldNotEqual, DefaultTRPCLogger)
		trpcLogger = getLogger("trpc.mysql.customLogger.db2")
		So(trpcLogger, ShouldEqual, DefaultTRPCLogger)
	})
}
