package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/plugin"
)

func TestPlugin(t *testing.T) {
	bts := `
plugins:
  database:
    kafka:
      max_request_size: 104857600
      max_response_size: 104857600
      rewrite_log: true
`

	cfg := trpc.Config{}
	err := yaml.Unmarshal([]byte(bts), &cfg)
	assert.Nil(t, err)
	var yamlNode *yaml.Node
	if configP, ok := cfg.Plugins[pluginType]; ok {
		if node, ok := configP[pluginName]; ok {
			yamlNode = &node
		}
	}

	p := &Plugin{}
	err = p.Setup(pluginName, &plugin.YamlNodeDecoder{Node: yamlNode})
	assert.Nil(t, err)
}

func TestLogReWriter(t *testing.T) {
	l := LogReWriter{}
	l.Print("print")
	l.Printf("printf %+v", "test")
	l.Println("println", "test")
	assert.Nil(t, nil)
}
