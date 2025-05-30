package gorm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"trpc.group/trpc-go/trpc-go/log"
)

// Colors color codes.
const (
	reset       = "\033[0m"
	red         = "\033[31m"
	green       = "\033[32m"
	yellow      = "\033[33m"
	blue        = "\033[34m"
	magenta     = "\033[35m"
	cyan        = "\033[36m"
	white       = "\033[37m"
	blueBold    = "\033[34;1m"
	magentaBold = "\033[35;1m"
	redBold     = "\033[31;1m"
	yellowBold  = "\033[33;1m"
)

// TRPCLogger implements the Gorm logger.Interface.
type TRPCLogger struct {
	maxSqlSize                          int
	config                              logger.Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

// NewTRPCLogger creates a TRPCLogger based on the configuration parameters.
func NewTRPCLogger(config logger.Config) *TRPCLogger {
	var (
		infoStr      = "%s\n[info] "
		warnStr      = "%s\n[warn] "
		errStr       = "%s\n[error] "
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)

	if config.Colorful {
		infoStr = green + "%s\n" + reset + green + "[info] " + reset
		warnStr = blueBold + "%s\n" + reset + magenta + "[warn] " + reset
		errStr = magenta + "%s\n" + reset + red + "[error] " + reset
		traceStr = green + "%s\n" + reset + yellow + "[%.3fms] " + blueBold +
			"[rows:%v]" + reset + " %s"
		traceWarnStr = green + "%s " + yellow + "%s\n" + reset + redBold +
			"[%.3fms] " + yellow + "[rows:%v]" + magenta + " %s" + reset
		traceErrStr = redBold + "%s " + magentaBold + "%s\n" + reset + yellow +
			"[%.3fms] " + blueBold + "[rows:%v]" + reset + " %s"
	}

	return &TRPCLogger{
		config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

// SetMaxSqlLength set the max length of log sql size and returns TRPCLogger.
func (p *TRPCLogger) SetMaxSqlLength(length int) logger.Interface {
	p.maxSqlSize = length
	return p
}

// LogMode changes the log level and returns a new TRPCLogger.
func (p *TRPCLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *p
	newLogger.config.LogLevel = level
	return &newLogger
}

// Info prints logs at the Info level.
func (p *TRPCLogger) Info(ctx context.Context, format string, args ...interface{}) {
	if p.config.LogLevel >= logger.Info {
		log.InfoContextf(ctx, p.infoStr+format, append([]interface{}{utils.FileWithLineNum()}, args...)...)
	}
}

// Warn logs warning level log.
func (p *TRPCLogger) Warn(ctx context.Context, format string, args ...interface{}) {
	if p.config.LogLevel >= logger.Warn {
		log.WarnContextf(ctx, p.warnStr+format, append([]interface{}{utils.FileWithLineNum()}, args...)...)
	}
}

// Error logs Error level log.
func (p *TRPCLogger) Error(ctx context.Context, format string, args ...interface{}) {
	if p.config.LogLevel >= logger.Error {
		log.ErrorContextf(ctx, p.errStr+format, append([]interface{}{utils.FileWithLineNum()}, args...)...)
	}
}

// Trace prints detailed SQL logs of the corresponding level based on the SQL execution situation
// and the configured log level.
func (p *TRPCLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if p.config.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && p.config.LogLevel >= logger.Error &&
		(!errors.Is(err, gorm.ErrRecordNotFound) || !p.config.IgnoreRecordNotFoundError):
		rawSql, rows := fc()
		sql := p.truncateSQL(rawSql)
		if rows == -1 {
			log.ErrorContextf(ctx, p.traceErrStr,
				utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			log.ErrorContextf(ctx, p.traceErrStr,
				utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > p.config.SlowThreshold && p.config.SlowThreshold != 0 && p.config.LogLevel >= logger.Warn:
		rawSql, rows := fc()
		sql := p.truncateSQL(rawSql)
		slowLog := fmt.Sprintf("SLOW SQL >= %v", p.config.SlowThreshold)
		if rows == -1 {
			log.WarnContextf(ctx, p.traceWarnStr,
				utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			log.WarnContextf(ctx, p.traceWarnStr,
				utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case p.config.LogLevel == logger.Info:
		rawSql, rows := fc()
		sql := p.truncateSQL(rawSql)
		if rows == -1 {
			log.InfoContextf(ctx, p.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			log.InfoContextf(ctx, p.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

func (p *TRPCLogger) truncateSQL(sql string) string {
	if p.maxSqlSize > 0 && len(sql) > p.maxSqlSize {
		return sql[:p.maxSqlSize-1] + " ..."
	}
	return sql
}
