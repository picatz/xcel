package xcel

import (
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ types.Adapter = TypeAdapter{}

type TypeAdapter map[reflect.Type]func(value any) ref.Val

func (ta TypeAdapter) NativeToValue(value any) ref.Val {
	for typ, fn := range ta {
		if reflect.TypeOf(value) == typ {
			return fn(value)
		}
	}
	return types.DefaultTypeAdapter.NativeToValue(value)
}

func NewTypeAdapter() TypeAdapter {
	return make(TypeAdapter)
}
