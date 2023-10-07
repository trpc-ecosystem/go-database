package bigcache

import (
	"errors"
	"fmt"
	"strconv"
)

// ToInterface converts a string type to the corresponding interface types such as bool, float, int, uint, error, etc.
func ToInterface(s string, to interface{}) (interface{}, error) {
	switch to.(type) {
	case bool:
		return strconv.ParseBool(s)
	case float64:
		return strconv.ParseFloat(s, 64)
	case float32:
		val, err := strconv.ParseFloat(s, 32)
		return float32(val), err
	case error:
		return errors.New(s), nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, err
	}
	switch to.(type) {
	case int:
		return int(i), nil
	case int64:
		return int64(i), nil
	case int32:
		return int32(i), nil
	case int16:
		return int16(i), nil
	case int8:
		return int8(i), nil
	case uint:
		return uint(i), nil
	case uint64:
		return uint64(i), nil
	case uint32:
		return uint32(i), nil
	case uint16:
		return uint16(i), nil
	case uint8:
		return uint8(i), nil
	default:
		return "", fmt.Errorf("unsupport type")
	}
}

// ToString converts bool, float, int, uint, error, and other types to string type.
func ToString(i interface{}) (string, error) {
	switch s := i.(type) {
	case bool:
		return strconv.FormatBool(s), nil
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(s), 'f', -1, 32), nil
	case int:
		return strconv.Itoa(s), nil
	case int64:
		return strconv.FormatInt(s, 10), nil
	case int32:
		return strconv.Itoa(int(s)), nil
	case int16:
		return strconv.FormatInt(int64(s), 10), nil
	case int8:
		return strconv.FormatInt(int64(s), 10), nil
	case uint:
		return strconv.FormatInt(int64(s), 10), nil
	case uint64:
		return strconv.FormatInt(int64(s), 10), nil
	case uint32:
		return strconv.FormatInt(int64(s), 10), nil
	case uint16:
		return strconv.FormatInt(int64(s), 10), nil
	case uint8:
		return strconv.FormatInt(int64(s), 10), nil
	case error:
		return s.Error(), nil
	default:
		return "", fmt.Errorf("unable to cast %#v of type %T to string", i, i)
	}
}
