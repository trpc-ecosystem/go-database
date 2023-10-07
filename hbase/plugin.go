package hbase

import (
	"sync"

	"trpc.group/trpc-go/trpc-go/plugin"
	"trpc.group/trpc-go/trpc-go/transport"
)

const (
	pluginType = "database"
	pluginName = "hbase"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin proxy 插件默认初始化, 用于加载hbase相关参数配置
type Plugin struct{}

// Type 获取插件类型
func (p *Plugin) Type() string {
	return pluginType
}

// Setup 安装插件
func (p *Plugin) Setup(name string, configDec plugin.Decoder) error {
	var config Config
	if err := configDec.Decode(&config); err != nil {
		return err
	}

	DefaultClientTransport = &ClientTransport{
		lock:    sync.RWMutex{},
		clients: map[string]*clientImp{},
		opts:    nil,
	}

	// 需要显式调用register，否则配置无法生效
	transport.RegisterClientTransport(pluginName, DefaultClientTransport)
	return nil
}
