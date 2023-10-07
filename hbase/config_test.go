package hbase

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_parseAddress(t *testing.T) {
	Convey("parseAddress", t, func() {
		addr := "127.0.0.1:2181,127.0.0.1:2181?zookeeperRoot=/root" +
			"&zookeeperTimeout=1000&regionLookupTimeout=1000&regionReadTimeout=1000&effectiveUser=root"
		expect := &Config{
			Addr:                "127.0.0.1:2181,127.0.0.1:2181",
			ZookeeperRoot:       "/root",
			ZookeeperTimeout:    1000,
			RegionLookupTimeout: 1000,
			RegionReadTimeout:   1000,
			EffectiveUser:       "root",
		}

		conf, err := parseAddress(addr)
		So(err, ShouldBeNil)
		So(conf, ShouldResemble, expect)

		Convey("no?-err", func() {
			addr := "127.0.0.1:2181,127.0.0.1:2181"
			conf, err := parseAddress(addr)
			So(err, ShouldNotBeNil)
			So(conf, ShouldBeNil)
		})

		Convey("uriNotValid-err", func() {
			addr := "127.0.0.1:2181,127.0.0.1:2181?x%x"
			conf, err := parseAddress(addr)
			t.Log(err)
			So(err, ShouldNotBeNil)
			So(conf, ShouldBeNil)
		})

		Convey("zookeeperTimeout-err", func() {
			addr := "127.0.0.1:2181,127.0.0.1:2181?zookeeperRoot=/root" +
				"&zookeeperTimeout=xxx&regionLookupTimeout=1000&regionReadTimeout=1000&effectiveUser=root"
			conf, err := parseAddress(addr)
			So(err, ShouldNotBeNil)
			So(conf, ShouldBeNil)
		})

		Convey("regionLookupTimeout-err", func() {
			addr := "127.0.0.1:2181,127.0.0.1:2181?zookeeperRoot=/root" +
				"&zookeeperTimeout=0&regionLookupTimeout=xxx&regionReadTimeout=1000&effectiveUser=root"
			conf, err := parseAddress(addr)
			So(err, ShouldNotBeNil)
			So(conf, ShouldBeNil)
		})

		Convey("regionReadTimeout-err", func() {
			addr := "127.0.0.1:2181,127.0.0.1:2181?zookeeperRoot=/root" +
				"&zookeeperTimeout=0&regionLookupTimeout=0&regionReadTimeout=xxx&effectiveUser=root"
			conf, err := parseAddress(addr)
			So(err, ShouldNotBeNil)
			So(conf, ShouldBeNil)
		})
	})

}
