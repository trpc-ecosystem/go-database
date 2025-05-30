package gorm

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/logger"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestGetGormPluginType(t *testing.T) {
	Convey("Get gorm plugin type", t, func() {
		sqlPlugin := new(Plugin)
		So(sqlPlugin.Type(), ShouldEqual, pluginType)
	})
}

func TestGormPluginSetupCompleteConfigs(t *testing.T) {
	Convey("Gorm plugin setup complete global config and complete service configs", t, func() {
		mp := &Plugin{}
		Convey("Config decode fail", func() {
			err := mp.Setup(pluginName, &plugin.YamlNodeDecoder{Node: nil})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "yaml node empty")
		})
		Convey("Setup success", func() {
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
            max_sql_size: 1024
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

func TestGormPluginSetupDefaultConfigs(t *testing.T) {
	Convey("Gorm plugin setup default global config and default service configs", t, func() {
		mp := &Plugin{}
		Convey("Setup success", func() {
			var bts = `
plugins:
  database:
    gorm:
      service:
        - name: trpc.mysql.test.db1
        - name: trpc.mysql.test.db2
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
					MaxIdle:     defaultMaxIdle,
					MaxOpen:     defaultMaxOpen,
					MaxLifetime: defaultMaxLifetime,
				},
				PoolConfigs: map[string]PoolConfig{
					"trpc.mysql.test.db1": {
						MaxIdle:     defaultMaxIdle,
						MaxOpen:     defaultMaxOpen,
						MaxLifetime: defaultMaxLifetime,
					},
					"trpc.mysql.test.db2": {
						MaxIdle:     defaultMaxIdle,
						MaxOpen:     defaultMaxOpen,
						MaxLifetime: defaultMaxLifetime,
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

func TestGormPluginSetupEffectOrder_P0(t *testing.T) {
	Convey("Service configs use the global config when there are no local configs", t, func() {
		mp := &Plugin{}
		Convey("Setup success", func() {
			var bts = `
plugins:
  database:
    gorm:
      max_idle: 20
      max_open: 100
      max_lifetime: 180
      service:
        - name: trpc.mysql.test.db1
        - name: trpc.mysql.test.db2
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
						MaxIdle:     20,
						MaxOpen:     100,
						MaxLifetime: 180 * time.Millisecond,
					},
					"trpc.mysql.test.db2": {
						MaxIdle:     20,
						MaxOpen:     100,
						MaxLifetime: 180 * time.Millisecond,
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

func TestGormPluginSetupEffectOrder_P1(t *testing.T) {
	Convey("Service local configs override default global config", t, func() {
		mp := &Plugin{}
		Convey("Setup success", func() {
			var bts = `
plugins:
  database:
    gorm:
      service:
        - name: trpc.mysql.test.db1
          max_idle: 20
          max_open: 300
          max_lifetime: 4000
        - name: trpc.mysql.test.db2
          max_idle: 50
          max_open: 600
          max_lifetime: 7000
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
					MaxIdle:     defaultMaxIdle,
					MaxOpen:     defaultMaxOpen,
					MaxLifetime: defaultMaxLifetime,
				},
				PoolConfigs: map[string]PoolConfig{
					"trpc.mysql.test.db1": {
						MaxIdle:     20,
						MaxOpen:     300,
						MaxLifetime: 4000 * time.Millisecond,
					},
					"trpc.mysql.test.db2": {
						MaxIdle:     50,
						MaxOpen:     600,
						MaxLifetime: 7000 * time.Millisecond,
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

func TestGormPluginSetupEffectOrder_P2(t *testing.T) {
	Convey("Local config 1 use default global config and local config 2 override default global config", t, func() {
		mp := &Plugin{}
		Convey("Setup success", func() {
			var bts = `
plugins:
  database:
    gorm:
      max_idle: 0
      max_open: 0
      max_lifetime: 0
      service:
        - name: trpc.mysql.test.db1
          max_idle: 0
          max_open: 0
        - name: trpc.mysql.test.db2
          max_idle: 50
          max_open: 600
          max_lifetime: 7000
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
					MaxIdle:     defaultMaxIdle,
					MaxOpen:     defaultMaxOpen,
					MaxLifetime: defaultMaxLifetime,
				},
				PoolConfigs: map[string]PoolConfig{
					"trpc.mysql.test.db1": {
						MaxIdle:     defaultMaxIdle,
						MaxOpen:     defaultMaxOpen,
						MaxLifetime: defaultMaxLifetime,
					},
					"trpc.mysql.test.db2": {
						MaxIdle:     50,
						MaxOpen:     600,
						MaxLifetime: 7000 * time.Millisecond,
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
