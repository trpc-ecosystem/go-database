package hbase

import (
	"testing"

	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go/plugin"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPlugin_Setup(t *testing.T) {
	p := &Plugin{}
	Convey("Plugin.Setup", t, func() {
		var conf yaml.Node
		_ = yaml.Unmarshal([]byte(""), conf)

		err := p.Setup(pluginName, &plugin.YamlNodeDecoder{Node: &conf})
		So(err, ShouldBeNil)
	})
}

func TestPlugin_Type(t *testing.T) {
	p := &Plugin{}
	Convey("Plugin.Type", t, func() {
		So(p.Type(), ShouldEqual, pluginType)
	})
}
