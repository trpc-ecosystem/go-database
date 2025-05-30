package gorm

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm/logger"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
)

//go:generate mockgen -destination=log.mock.go -package=gorm trpc.group/trpc-go/trpc-go/log Logger

func TestUnit_Log_LogMode_P0(t *testing.T) {
	Convey("TestUnit_Log_LogMode_P0", t, func() {
		testLog := NewTRPCLogger(logger.Config{LogLevel: logger.Info})
		afterLogMode := testLog.LogMode(logger.Error)
		So(afterLogMode.(*TRPCLogger).config.LogLevel, ShouldEqual, logger.Error)
	})
}

func TestUnit_Log_Colorful(t *testing.T) {
	Convey("TestUnit_Log_Colorful", t, func() {
		testLog := NewTRPCLogger(logger.Config{Colorful: true})
		So(testLog.infoStr, ShouldEqual, green+"%s\n"+reset+green+"[info] "+reset)
	})
}

func TestUnit_Log_Info_P0(t *testing.T) {
	Convey("TestUnit_Log_Info_P0", t, func() {
		infoLog := NewTRPCLogger(logger.Config{LogLevel: logger.Info})
		calledTimes := 0

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLogger := NewMockLogger(mockCtrl)

		ctx := trpc.BackgroundContext()
		codec.Message(ctx).WithLogger(mockLogger)

		mockLogger.EXPECT().
			Infof(gomock.Any(), gomock.Any()).
			DoAndReturn(func(format string, args ...interface{}) {
				So(format, ShouldEqual, infoLog.infoStr+"test log: %d")
				So(args[1], ShouldEqual, 1)
				calledTimes++
			})

		infoLog.Info(ctx, "test log: %d", 1)
		So(calledTimes, ShouldEqual, 1)
	})
}

func TestUnit_Log_Warn_P0(t *testing.T) {
	Convey("TestUnit_Log_Warn_P0", t, func() {
		silentLog := NewTRPCLogger(logger.Config{LogLevel: logger.Silent})
		warnLog := NewTRPCLogger(logger.Config{LogLevel: logger.Warn})
		calledTimes := 0

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLogger := NewMockLogger(mockCtrl)

		ctx := trpc.BackgroundContext()
		codec.Message(ctx).WithLogger(mockLogger)

		mockLogger.EXPECT().
			Warnf(gomock.Any(), gomock.Any()).
			DoAndReturn(func(format string, args ...interface{}) {
				So(format, ShouldEqual, warnLog.warnStr+"test log: %d")
				So(args[1], ShouldEqual, 1)
				calledTimes++
			}).AnyTimes()

		silentLog.Warn(ctx, "test log")
		So(calledTimes, ShouldEqual, 0)

		warnLog.Warn(ctx, "test log: %d", 1)
		So(calledTimes, ShouldEqual, 1)
	})
}

func TestUnit_Log_Error_P0(t *testing.T) {
	Convey("TestUnit_Log_Error_P0", t, func() {
		silentLog := NewTRPCLogger(logger.Config{LogLevel: logger.Silent})
		errorLog := NewTRPCLogger(logger.Config{LogLevel: logger.Error})
		calledTimes := 0

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLogger := NewMockLogger(mockCtrl)

		ctx := trpc.BackgroundContext()
		codec.Message(ctx).WithLogger(mockLogger)

		mockLogger.EXPECT().
			Errorf(gomock.Any(), gomock.Any()).
			DoAndReturn(func(format string, args ...interface{}) {
				So(format, ShouldEqual, errorLog.errStr+"test log: %d")
				So(args[1], ShouldEqual, 1)
				calledTimes++
			}).AnyTimes()

		silentLog.Error(ctx, "test log")
		So(calledTimes, ShouldEqual, 0)

		errorLog.Error(ctx, "test log: %d", 1)
		So(calledTimes, ShouldEqual, 1)
	})
}

func TestUnit_Log_Trace_P0(t *testing.T) {
	Convey("TestUnit_Log_Trace_P0", t, func() {
		silentLog := NewTRPCLogger(logger.Config{LogLevel: logger.Silent})
		infoLog := NewTRPCLogger(logger.Config{LogLevel: logger.Info})
		infoLog.SetMaxSqlLength(2)
		calledTimes := 0

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLogger := NewMockLogger(mockCtrl)

		ctx := trpc.BackgroundContext()
		codec.Message(ctx).WithLogger(mockLogger)

		mockLogger.EXPECT().
			Infof(gomock.Any(), gomock.Any()).
			DoAndReturn(func(format string, args ...interface{}) {
				So(format, ShouldEqual, infoLog.traceStr)
				if calledTimes == 0 {
					So(args[2], ShouldEqual, 7)
					So(args[3], ShouldEqual, "t ...")
				} else {
					So(args[2], ShouldEqual, "-")
					So(args[3], ShouldEqual, "")
				}
				calledTimes++
			}).AnyTimes()

		silentLog.Trace(ctx, time.Now(), func() (string, int64) { return "this is sql", 7 }, nil)
		So(calledTimes, ShouldEqual, 0)

		infoLog.Trace(ctx, time.Now(), func() (string, int64) { return "this is sql", 7 }, nil)
		So(calledTimes, ShouldEqual, 1)

		infoLog.Trace(ctx, time.Now(), func() (string, int64) { return "", -1 }, nil)
		So(calledTimes, ShouldEqual, 2)

		warnLog := NewTRPCLogger(logger.Config{LogLevel: logger.Warn, SlowThreshold: time.Second})
		mockLogger.EXPECT().
			Warnf(gomock.Any(), gomock.Any()).
			DoAndReturn(func(format string, args ...interface{}) {
				So(format, ShouldEqual, warnLog.traceWarnStr)
				So(args[1], ShouldEqual, "SLOW SQL >= 1s")
				if calledTimes == 2 {
					So(args[3], ShouldEqual, 7)
					So(args[4], ShouldEqual, "this is sql")
				} else {
					So(args[3], ShouldEqual, "-")
					So(args[4], ShouldEqual, "")
				}
				calledTimes++
			}).AnyTimes()

		warnLog.Trace(ctx, time.Now().Add(-1*time.Second), func() (string, int64) { return "this is sql", 7 }, nil)
		So(calledTimes, ShouldEqual, 3)

		warnLog.Trace(ctx, time.Now().Add(-1*time.Second), func() (string, int64) { return "", -1 }, nil)
		So(calledTimes, ShouldEqual, 4)

		errorLog := NewTRPCLogger(logger.Config{LogLevel: logger.Error, IgnoreRecordNotFoundError: true})
		mockLogger.EXPECT().
			Errorf(gomock.Any(), gomock.Any()).
			DoAndReturn(func(format string, args ...interface{}) {
				So(format, ShouldEqual, warnLog.traceWarnStr)
				So(format, ShouldEqual, errorLog.traceErrStr)
				So(args[1], ShouldBeError, errors.New("this is err"))
				if calledTimes == 4 {
					So(args[3], ShouldEqual, 7)
					So(args[4], ShouldEqual, "this is sql")
				} else {
					So(args[3], ShouldEqual, "-")
					So(args[4], ShouldEqual, "")
				}
				calledTimes++
			}).AnyTimes()

		errorLog.Trace(ctx, time.Now(), func() (string, int64) { return "this is sql", 7 }, errors.New("this is err"))
		So(calledTimes, ShouldEqual, 5)

		errorLog.Trace(ctx, time.Now(), func() (string, int64) { return "", -1 }, errors.New("this is err"))
		So(calledTimes, ShouldEqual, 6)
	})
}
