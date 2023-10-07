package hbase

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"trpc.group/trpc-go/trpc-go/log"
)

func Test_logger_Write(t *testing.T) {
	l := new(logger)
	p := []byte("xxx")
	Convey("logger.Write", t, func() {
		Convey("trace", func() {
			SetLogLevel(log.LevelTrace)
			n, err := l.Write(p)
			So(n, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})
		Convey("debug", func() {
			SetLogLevel(log.LevelDebug)
			n, err := l.Write(p)
			So(n, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})
		Convey("info", func() {
			SetLogLevel(log.LevelInfo)
			n, err := l.Write(p)
			So(n, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})
		Convey("warn", func() {
			SetLogLevel(log.LevelWarn)
			n, err := l.Write(p)
			So(n, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})
		Convey("error", func() {
			SetLogLevel(log.LevelError)
			n, err := l.Write(p)
			So(n, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})
	})
}
