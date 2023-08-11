package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/plugin"

	. "github.com/smartystreets/goconvey/convey"
)

// TestUnit_MysqlPlugin_Type_P0 MysqlPlugin.Type test case.
func TestUnit_MysqlPlugin_Type_P0(t *testing.T) {
	Convey("TestUnit_MysqlPlugin_Type_P0", t, func() {
		mysqlPlugin := new(Plugin)
		So(mysqlPlugin.Type(), ShouldEqual, pluginType)
	})
}

// TestUnit_MysqlPlugin_Setup_P0 MysqlPlugin.Setup test case
func TestUnit_MysqlPlugin_Setup_P0(t *testing.T) {
	Convey("TestUnit_MysqlPlugin_Setup_P0", t, func() {
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
    mysql:
      max_idle: 20
      max_open: 100
      max_lifetime: 180
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
			err = mp.Setup(pluginName, &plugin.YamlNodeDecoder{Node: yamlNode})
			So(err, ShouldBeNil)
		})
	})
}
