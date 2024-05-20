package mongodb

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/plugin"
)

// TestUnit_MongoPlugin_Type_P0 MongoPlugin.Type is test case.
func TestUnit_MongoPlugin_Type_P0(t *testing.T) {
	Convey("TestUnit_MongoPlugin_Type_P0", t, func() {
		mongoPlugin := new(MongoPlugin)
		So(mongoPlugin.Type(), ShouldEqual, pluginType)
	})
}

// TestUnit_MongoPlugin_Setup_P0 MongoPlugin.Setup is test case.
func TestUnit_MongoPlugin_Setup_P0(t *testing.T) {
	Convey("TestUnit_MongoPlugin_Setup_P0", t, func() {
		mp := &MongoPlugin{}
		Convey("Config Decode Fail", func() {
			err := mp.Setup(pluginName, &plugin.YamlNodeDecoder{Node: nil})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "yaml node empty")
		})
		Convey("Setup Success", func() {
			var bts = `
plugins:
  database:
    mongodb:
      min_open: 20
      max_open: 100
      max_idle_time: 1s
      read_preference: secondar
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

// TestUnit_MongoPlugin_Setup_Fail MongoPlugin.Setup is typo test case.
func TestUnit_MongoPlugin_Setup_Fail(t *testing.T) {

	Convey("TestUnit_MongoPlugin_Setup_Fail", t, func() {
		mp := &MongoPlugin{}
		Convey("Config Decode Fail", func() {
			err := mp.Setup(pluginName, &plugin.YamlNodeDecoder{Node: nil})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "yaml node empty")
		})
		Convey("Setup Success", func() {
			var bts = `
plugins:
  database:
    mongodb:
      min_open: 20
      max_open: 100
      max_idle_time: 1s
      read_preference: secondary
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

// TestUnit_MongoPlugin_Setup_New_READF_Failed MongoPlugin.Setup is test case.
func TestUnit_MongoPlugin_Setup_New_READF_Failed(t *testing.T) {
	defer gomonkey.ApplyFunc(readpref.New,
		func(mode readpref.Mode, opts ...readpref.Option) (*readpref.ReadPref, error) {
			return nil, errors.New("test fail")
		},
	).Reset()

	Convey("TestUnit_MongoPlugin_Setup_P0", t, func() {
		mp := &MongoPlugin{}
		Convey("Config Decode Fail", func() {
			err := mp.Setup(pluginName, &plugin.YamlNodeDecoder{Node: nil})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "yaml node empty")
		})
		Convey("Setup Success", func() {
			var bts = `
plugins:
  database:
    mongodb:
      min_open: 20
      max_open: 100
      max_idle_time: 1s
      read_preference: secondary
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
