package phoenix

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/vektah/gqlgen/graphql"
)

// MarshalUint implements a Uint value
func MarshalUint(t uint) graphql.Marshaler {
	return MarshalUint64(uint64(t))
}

// UnmarshalUint implements a Uint value
func UnmarshalUint(v interface{}) (uint, error) {
	i, err := UnmarshalUint64(v)
	return uint(i), err
}

// MarshalUint64 implements a Uint64 value
func MarshalUint64(t uint64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, err := io.WriteString(w, strconv.FormatUint(t, 10))
		if err != nil {
			return
		}
	})
}

// UnmarshalUint64 implements a Uint64 value
func UnmarshalUint64(v interface{}) (uint64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseUint(t, 10, 64)
	case int:
		return uint64(t), nil
	case int64:
		return uint64(t), nil
	case json.Number:
		i, err := t.Int64()
		return uint64(i), err
	case float64:
		return uint64(t), nil
	}

	return 0, fmt.Errorf("unable to unmarshal uint64: %#v %T", v, v)
}

// MarshalUint32 implements a Uint32 value
func MarshalUint32(t uint32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, err := io.WriteString(w, strconv.FormatUint(uint64(t), 10))
		if err != nil {
			return
		}
	})
}

// UnmarshalUint32 implements a Uint32 value
func UnmarshalUint32(v interface{}) (uint32, error) {
	switch t := v.(type) {
	case string:
		u, err := strconv.ParseUint(t, 10, 32)
		return uint32(u), err
	case int:
		return uint32(t), nil
	case int64:
		return uint32(t), nil
	case json.Number:
		i, err := t.Int64()
		return uint32(i), err
	case float64:
		return uint32(t), nil
	}

	return 0, fmt.Errorf("unable to unmarshal uint32: %#v %T", v, v)
}

// MarshalInt implements a Int value
func MarshalInt(t int) graphql.Marshaler {
	return MarshalInt64(int64(t))
}

// UnmarshalInt implements a Int value
func UnmarshalInt(v interface{}) (int, error) {
	i, err := UnmarshalInt64(v)
	return int(i), err
}

// MarshalInt64 implements a Int64 value
func MarshalInt64(t int64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, err := io.WriteString(w, strconv.FormatInt(t, 10))
		if err != nil {
			return
		}
	})
}

// UnmarshalInt64 implements a Int64 value
func UnmarshalInt64(v interface{}) (int64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseInt(t, 10, 64)
	case int:
		return int64(t), nil
	case int64:
		return int64(t), nil
	case json.Number:
		i, err := t.Int64()
		return int64(i), err
	case float64:
		return int64(t), nil
	}

	return 0, fmt.Errorf("unable to unmarshal uint64: %#v %T", v, v)
}

// MarshalInt32 implements a Int32 value
func MarshalInt32(t int32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, err := io.WriteString(w, strconv.FormatInt(int64(t), 10))
		if err != nil {
			return
		}
	})
}

// UnmarshalInt32 implements a Int32 value
func UnmarshalInt32(v interface{}) (int32, error) {
	switch t := v.(type) {
	case string:
		u, err := strconv.ParseInt(t, 10, 32)
		return int32(u), err
	case int:
		return int32(t), nil
	case int64:
		return int32(t), nil
	case json.Number:
		i, err := t.Int64()
		return int32(i), err
	case float64:
		return int32(t), nil
	}

	return 0, fmt.Errorf("unable to unmarshal uint32: %#v %T", v, v)
}
