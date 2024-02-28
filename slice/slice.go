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

type VisibleCheckerMap[T comparable, U any] interface {
	IsVisible(U) bool
	Map() map[T]U
	MarshalJSON() ([]byte, error)
}

type DefaultVisibleCheckerMap[T comparable, U any] struct {
	m       map[T]U
	checker func(U) bool
}

func (d DefaultVisibleCheckerMap[T, U]) IsVisible(u U) bool {
	return d.checker(u)
}

func (d DefaultVisibleCheckerMap[T, U]) Map() map[T]U {
	return d.m
}

func (d DefaultVisibleCheckerMap[T, U]) MarshalJSON() ([]byte, error) {
	return MarshalMapAsSliceJSON(d)
}

func NewVisibleCheckerMap[T comparable, U any](m map[T]U, f func(U) bool) DefaultVisibleCheckerMap[T, U] {
	if f == nil {
		f = func(u U) bool { return true }
	}
	return DefaultVisibleCheckerMap[T, U]{m, f}
}

func MarshalMapAsSliceJSON[T comparable, U any](m VisibleCheckerMap[T, U], itemSize ...int) ([]byte, error) {
	size := 64
	if len(itemSize) > 0 {
		size = itemSize[0]
	}
	var buf = bytes.NewBuffer(make([]byte, 0, size*len(m.Map())))
	encoder := json.NewEncoder(buf)
	buf.WriteByte('[')
	first := true
	for _, o := range m.Map() {
		if !m.IsVisible(o) {
			continue
		}
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
