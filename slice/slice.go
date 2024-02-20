package slice

import (
	"bytes"
	"encoding/json"
)

func MapToSlice[T comparable, U any](m map[T]U) []U {
	result := make([]U, 0, len(m))
	for _, o := range m {
		result = append(result, o)
	}
	return result
}

func SliceToMap[T comparable, U any](s []T, f func(T) U) map[T]U {
	result := make(map[T]U, len(s))
	for _, o := range s {
		result[o] = f(o)
	}
	return result
}

func MarshalMapAsSliceJSON[T comparable, U any](m map[T]U, itemSize ...int) ([]byte, error) {
	size := 64
	if len(itemSize) > 0 {
		size = itemSize[0]
	}
	var buf = bytes.NewBuffer(make([]byte, 0, size*len(m)))
	encoder := json.NewEncoder(buf)
	buf.WriteByte('[')
	first := true
	for _, o := range m {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		if err := encoder.Encode(o); err != nil {
			return nil, err
		}
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}
