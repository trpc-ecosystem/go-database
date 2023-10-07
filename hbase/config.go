package hbase

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Config HBase配置
// hbase://zk_addr?zookeeperRoot=/xx/xxx&zookeeperTimeout=100&regionLookupTimeout=100
// &regionReadTimeout=100&effectiveUser=root
type Config struct {
	Addr                string `yaml:"addr"`
	ZookeeperRoot       string `yaml:"zookeeperRoot"`
	ZookeeperTimeout    int    `yaml:"zookeeperTimeout"`    // 单位毫秒
	RegionLookupTimeout int    `yaml:"regionLookupTimeout"` // 单位毫秒
	RegionReadTimeout   int    `yaml:"regionReadTimeout"`   // 单位毫秒
	EffectiveUser       string `yaml:"effectiveUser"`       // 访问hbase的用户名
}

func parseAddress(address string) (*Config, error) {
	conf := &Config{}
	pos := strings.Index(address, "?")
	if pos == -1 {
		return nil, fmt.Errorf("not find ? in address[%v]", address)
	}

	// 解析后面的参数
	uri, err := url.ParseQuery(address[pos+1:])
	if err != nil {
		return nil, fmt.Errorf("uri[%v] parse err: %v", address[pos+1:], err)
	}

	// 参数赋值
	conf.Addr = address[:pos]
	conf.ZookeeperRoot = uri.Get("zookeeperRoot")

	zookeeperTimeout := uri.Get("zookeeperTimeout")
	conf.ZookeeperTimeout, err = strconv.Atoi(zookeeperTimeout)
	if err != nil {
		return nil, fmt.Errorf("zookeeperTimeout[%v] parse int err: %v", zookeeperTimeout, err)
	}

	regionLookupTimeout := uri.Get("regionLookupTimeout")
	conf.RegionLookupTimeout, err = strconv.Atoi(regionLookupTimeout)
	if err != nil {
		return nil, fmt.Errorf("regionLookupTimeout[%v] parse int err: %v", regionLookupTimeout, err)
	}

	regionReadTimeout := uri.Get("regionReadTimeout")
	conf.RegionReadTimeout, err = strconv.Atoi(regionReadTimeout)
	if err != nil {
		return nil, fmt.Errorf("regionReadTimeout[%v] parse int err: %v", regionReadTimeout, err)
	}

	conf.EffectiveUser = uri.Get("effectiveUser")

	return conf, nil

}
