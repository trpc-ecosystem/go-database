package bigcache_test

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	gbigcache "github.com/allegro/bigcache/v3"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"

	"trpc.group/trpc-go/trpc-database/bigcache"
)

// go test -test.bench=".*"
func BenchmarkCache(b *testing.B) {
	cache, _ := bigcache.New(
		// Set the number of shards, default is 256.
		bigcache.WithShards(1024),
		// Set the life window, default is 7 days.
		bigcache.WithLifeWindow(time.Hour),
		// Set the clean window, default is 1 minute.
		bigcache.WithCleanWindow(-1*time.Second),
		// Set the maximum number of entries in the window, default is 200,000.
		// A smaller value can reduce memory usage during initialization, and it's only used during initialization.
		bigcache.WithMaxEntriesInWindow(50000),
		// Set the maximum entry size in bytes, default is 30.
		// A smaller value can reduce memory usage during initialization, and it's only used during initialization.
		bigcache.WithMaxEntrySize(20),
	)

	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			i++
			// The loop body is executed b.N times total across all goroutines.
			key := fmt.Sprintf("key:%d", i)
			val := []byte(fmt.Sprintf("val:%d", i))
			// write data
			err := cache.Set(key, val)
			if err != nil {
				b.Errorf("cache set fail:%v", err)
				return
			}

			// get data
			data, err := cache.GetBytes(key)
			if err != nil {
				b.Errorf("cache get fail:%v", err)
				return
			}

			if string(data) != string(val) {
				b.Errorf("cache get data not equal")
				return
			}

		}

	})
}

type args struct {
	key               string
	val               interface{}
	serializationType []int
}

// dummyStruct dummy struct
type dummyStruct struct {
	Key   string
	Value string
}

// test val
var (
	valString = &args{
		key:               "vString",
		val:               "s1",
		serializationType: nil,
	}
	valByte = &args{
		key:               "vByte",
		val:               []byte{0, 1, 0, 1},
		serializationType: nil,
	}
	valBool = &args{
		key:               "vBool",
		val:               true,
		serializationType: nil,
	}
	valFloat64 = &args{
		key:               "vFloat64",
		val:               math.MaxFloat64,
		serializationType: nil,
	}
	valFloat32 = &args{
		key:               "vFloat32",
		val:               float32(math.MaxFloat32),
		serializationType: nil,
	}
	valInt = &args{
		key:               "vInt",
		val:               int(-1),
		serializationType: nil,
	}
	valInt64 = &args{
		key:               "vInt64",
		val:               int64(-math.MaxInt64),
		serializationType: nil,
	}
	valInt32 = &args{
		key:               "vInt32",
		val:               int32(-math.MaxInt32),
		serializationType: nil,
	}
	valInt16 = &args{
		key:               "vInt16",
		val:               int16(-math.MaxInt16),
		serializationType: nil,
	}
	valInt8 = &args{
		key:               "vInt8",
		val:               int8(-math.MaxInt8),
		serializationType: nil,
	}
	valUint = &args{
		key:               "vUint",
		val:               uint(1),
		serializationType: nil,
	}
	valUint64 = &args{
		key:               "vUint64",
		val:               uint64(math.MaxUint64),
		serializationType: nil,
	}
	valUint32 = &args{
		key:               "vUint32",
		val:               uint32(math.MaxInt32),
		serializationType: nil,
	}
	valUint16 = &args{
		key:               "vUint16",
		val:               uint16(math.MaxInt16),
		serializationType: nil,
	}
	valUint8 = &args{
		key:               "vUint8",
		val:               uint8(math.MaxInt8),
		serializationType: nil,
	}
	valStruct = &args{
		key:               "vStruct",
		val:               dummyStruct{Key: "a", Value: "b"},
		serializationType: []int{codec.SerializationTypeJSON},
	}
	valStructWithoutSerialization = &args{
		key:               "valStructWithoutSerialization",
		val:               dummyStruct{Key: "a", Value: "b"},
		serializationType: nil,
	}
	valWrong = &args{
		key: "map",
		val: map[string]interface{}{
			"a": "a",
			"b": 1,
		},
		serializationType: nil,
	}
	valError = &args{
		key:               "error",
		val:               errors.New("error test"),
		serializationType: nil,
	}
)

func TestBigCache_GetSet(t *testing.T) {
	type fields struct {
		bc *gbigcache.BigCache
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantValue interface{}
	}{
		{name: "setString", args: *valString, wantErr: false, wantValue: valString.val},
		{name: "setByte", args: *valByte, wantErr: false, wantValue: valByte.val},
		{name: "setBool", args: *valBool, wantErr: false, wantValue: valBool.val},
		{name: "setFloat32", args: *valFloat32, wantErr: false, wantValue: valFloat32.val},
		{name: "setFloat64", args: *valFloat64, wantErr: false, wantValue: valFloat64.val},
		{name: "setInt", args: *valInt, wantErr: false, wantValue: valInt.val},
		{name: "setInt8", args: *valInt8, wantErr: false, wantValue: valInt8.val},
		{name: "setInt16", args: *valInt16, wantErr: false, wantValue: valInt16.val},
		{name: "setInt32", args: *valInt32, wantErr: false, wantValue: valInt32.val},
		{name: "setInt64", args: *valInt64, wantErr: false, wantValue: valInt64.val},
		{name: "setUint", args: *valUint, wantErr: false, wantValue: valUint.val},
		{name: "setUint8", args: *valUint8, wantErr: false, wantValue: valUint8.val},
		{name: "setUint16", args: *valUint16, wantErr: false, wantValue: valUint16.val},
		{name: "setUint32", args: *valUint32, wantErr: false, wantValue: valUint32.val},
		{name: "setUint64", args: *valUint64, wantErr: false, wantValue: valUint64.val},
		{name: "valStruct", args: *valStruct, wantErr: false, wantValue: valStruct.val},
		{name: "valStructWithoutSerialization",
			args: *valStructWithoutSerialization, wantErr: true, wantValue: nil},
		{name: "setWrong", args: *valWrong, wantErr: true, wantValue: nil},
		{name: "setError", args: *valError, wantErr: false, wantValue: valError.val},
	}
	c, _ := bigcache.New([]bigcache.Option{
		bigcache.WithShards(512),
		bigcache.WithLifeWindow(time.Hour),
		bigcache.WithCleanWindow(time.Hour),
		bigcache.WithMaxEntriesInWindow(1000),
		bigcache.WithHardMaxCacheSize(1024),
		bigcache.WithVerbose(true),
		bigcache.WithStatsEnabled(true),
	}...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set val
			if err := c.Set(tt.args.key, tt.args.val, tt.args.serializationType...); (err != nil) != tt.wantErr {
				t.Errorf("%s: Set() error = %v, wantErr %v", tt.args.key, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			// get val
			bd, err := c.GetBytes(tt.args.key)
			if err != nil {
				t.Errorf("GetBytes() error = %v", err)
			}
			var value interface{}
			switch tt.args.val.(type) {
			case string:
				value = tt.args.val.(string)
			case []byte:
				value = bd
			case dummyStruct:
				var ds dummyStruct
				err = codec.Unmarshal(tt.args.serializationType[0], bd, &ds)
				assert.Nil(t, err)
				value = ds
			default:
				value, err = bigcache.ToInterface(string(bd), tt.args.val)
				if err != nil {
					t.Errorf("ToInterface() error = %v", err)
				}
			}
			if !reflect.DeepEqual(value, tt.wantValue) {
				t.Errorf("Get() value wrong value = %v", value)
				if value != tt.wantValue {
					t.Errorf("Get() value wrong value type = %v", reflect.TypeOf(value))
				}
			}
		})
	}
	_ = c.Close()
}

func TestBigCache_SetSerialize(t *testing.T) {
	c, _ := bigcache.New()
	type x struct {
		S string
		I int
		F float64
	}
	m := &x{
		"s", 1, 0.2,
	}
	err := c.Set("m", m, codec.SerializationTypeJSON)
	assert.Nil(t, err)
	m2 := new(x)
	err = c.Get("m", &m2, codec.SerializationTypeJSON)
	assert.Nil(t, err)
	assert.EqualValues(t, m, m2)
	m3 := new(x)
	err = c.Get("not_exist", &m3, codec.SerializationTypeJSON)
	assert.EqualValues(t, bigcache.ErrEntryNotFound, errs.Code(err))
	_ = c.Close()
}

func TestBigCache_Reset(t *testing.T) {
	c, _ := bigcache.New()
	err := c.Set("a", "1")
	assert.Nil(t, err)
	err = c.Set("b", "2")
	assert.Nil(t, err)
	err = c.Set("c", "3")
	assert.Nil(t, err)
	assert.EqualValues(t, 3, c.Len())
	err = c.Reset()
	assert.Nil(t, err)
	assert.Zero(t, c.Len())

	_ = c.Close()
}

func TestBigCache_Delete(t *testing.T) {
	c, err := bigcache.New()
	assert.Nil(t, err)
	err = c.Set("a", "1")
	assert.Nil(t, err)
	err = c.Set("b", "2")
	assert.Nil(t, err)
	err = c.Set("c", "3")
	assert.Nil(t, err)
	assert.EqualValues(t, 3, c.Len())
	err = c.Delete("a")
	assert.Nil(t, err)
	assert.EqualValues(t, 2, c.Len())
	_, err = c.GetBytes("a")
	assert.EqualValues(t, bigcache.ErrEntryNotFound, errs.Code(err))
	err = c.Delete("not_exist")
	assert.NotNil(t, err)
	assert.EqualValues(t, 2, c.Len())

	_ = c.Close()
}

func TestBigCache_Iterator(t *testing.T) {
	c, err := bigcache.New()
	assert.Nil(t, err)
	err = c.Set("1", 11)
	assert.Nil(t, err)
	err = c.Set("2", 22)
	assert.Nil(t, err)
	err = c.Set("3", 33)
	assert.Nil(t, err)
	iterator := c.Iterator()
	i := 3
	for iterator.SetNext() {
		assert.Greater(t, i, 0)
		e, err := iterator.Value()
		assert.Nil(t, err)
		assert.NotEmpty(t, e.Key())
		v, _ := strconv.ParseInt(string(e.Value()), 10, 64)
		assert.Greater(t, v, int64(10))
		i--
	}
	assert.EqualValues(t, 0, i)

	_ = c.Close()
}

func TestBigCache_GetBytesWithInfo(t *testing.T) {
	var removeReason gbigcache.RemoveReason
	c, err := bigcache.New(
		bigcache.WithLifeWindow(500*time.Millisecond),
		bigcache.WithLifeWindow(1*time.Second),
		bigcache.WithOnRemoveWithReasonCallback(func(key string, entry []byte, reason gbigcache.RemoveReason) {
			removeReason = reason
			return
		}),
	)
	// normal get
	assert.Nil(t, err)
	err = c.Set("1", "11")
	assert.Nil(t, err)
	b, rsp, err := c.GetBytesWithInfo("1")
	assert.Nil(t, err)
	assert.EqualValues(t, "11", string(b))
	assert.EqualValues(t, gbigcache.RemoveReason(0), rsp.EntryStatus)

	// expired
	err = c.Set("2", "22")
	assert.Nil(t, err)
	time.Sleep(2 * time.Second)
	b, rsp, err = c.GetBytesWithInfo("2")
	assert.Nil(t, err)
	assert.EqualValues(t, gbigcache.Expired, rsp.EntryStatus)

	// deleted
	err = c.Set("3", "33")
	assert.Nil(t, err)
	err = c.Delete("3")
	assert.Nil(t, err)
	b, rsp, err = c.GetBytesWithInfo("3")
	assert.EqualValues(t, bigcache.ErrEntryNotFound, errs.Code(err))
	assert.EqualValues(t, gbigcache.Deleted, removeReason)
}
