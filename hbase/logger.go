package hbase

import (
	"trpc.group/trpc-go/trpc-go/log"
)

type logger struct{}

var (
	loggerLevel   log.Level = log.LevelError
	defaultlogger           = new(logger)
)

// Write 写日志，重定向到 trpc-go 的log
func (l *logger) Write(p []byte) (n int, err error) {
	switch loggerLevel {
	case log.LevelTrace:
		log.Tracef("%v", string(p))
	case log.LevelDebug:
		log.Debugf("%v", string(p))
	case log.LevelInfo:
		log.Infof("%v", string(p))
	case log.LevelWarn:
		log.Warnf("%v", string(p))
	case log.LevelError:
		log.Errorf("%v", string(p))
	case log.LevelFatal:
		log.Fatalf("%v", string(p))
	default:
		// do nothing
	}

	return 0, nil
}

// SetLogLevel 设置log leve级别
func SetLogLevel(level log.Level) {
	loggerLevel = level
}
