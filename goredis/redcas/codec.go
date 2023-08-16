package redcas

import (
	"encoding"
	"encoding/binary"
	"encoding/json"
	"unsafe"

	"google.golang.org/protobuf/proto"
	goredis "trpc.group/trpc-go/trpc-database/goredis"
)

// Encode is to encode packet.
func Encode(value []byte, cas int64) []byte {
	raw := make([]byte, len(value)+8)
	binary.LittleEndian.PutUint64(raw, uint64(cas))
	copy(raw[8:], value)
	return raw
}

// Decode is to decode packet.
func Decode(raw []byte) ([]byte, int64, error) {
	if len(raw) < 8 {
		return nil, 0, goredis.ErrTypeMismatch
	}
	cas := int64(binary.LittleEndian.Uint64(raw))
	value := raw[8:]
	return value, cas, nil
}

// Marshal is type instantiation.
func Marshal(in interface{}) ([]byte, error) {
	switch out := in.(type) {
	case []byte:
		return out, nil
	case string:
		return StringToBytes(out), nil
	case proto.Message:
		return proto.Marshal(out)
	case encoding.BinaryMarshaler:
		return out.MarshalBinary()
	default:
		return nil, goredis.ErrTypeMismatch
	}
}

// Unmarshal is type deserialization.
func Unmarshal(in []byte, message interface{}) error {
	switch out := message.(type) {
	case *[]byte:
		*out = in
		return nil
	case *string:
		*out = BytesToString(in)
		return nil
	case proto.Message:
		return proto.Unmarshal(in, out)
	case encoding.BinaryUnmarshaler:
		return out.UnmarshalBinary(in)
	default:
		return goredis.ErrTypeMismatch
	}
}

// BytesToString converts byte slice to string.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes converts string to byte slice.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// MarshalJSONString is convenient for printing logs.
func MarshalJSONString(v interface{}) string {
	b, _ := json.Marshal(v)
	return BytesToString(b)
}
