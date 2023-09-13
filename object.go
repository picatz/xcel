package xcel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// Object is a CEL value wrapper for a Go value that
// can be used in expressions.
type Object[T any] struct {
	Raw T
}

// NewObject creates a new CEL value wrapper for a Go value
// that can be used in expressions.
func NewObject[T any](ta TypeAdapter, tp *TypeProvider, val T) (*Object[T], *types.Type) {
	return &Object[T]{Raw: val}, cel.ObjectType(reflect.TypeOf(val).String(), traits.ReceiverType)
}

// ConvertToNative converts the CEL value wrapper to a native Go value.
func (o *Object[T]) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if typeDesc == reflect.TypeOf(o.Raw) {
		return o.Raw, nil
	}
	return nil, fmt.Errorf("celval: type conversion error from '%s' to '%s'", o.Type(), typeDesc)
}

// ConvertToType converts the CEL value wrapper to a CEL value of the specified type.
func (o *Object[T]) ConvertToType(typeValue ref.Type) ref.Val {
	if typeValue == o.Type() {
		return o
	}
	return types.NewErr("celval: type conversion error from '%s' to '%s'", o.Type(), typeValue)
}

// Equal returns true if the CEL value wrapper is equal to the specified CEL value.
func (o *Object[T]) Equal(other ref.Val) ref.Val {
	if other, ok := other.(*Object[T]); ok {
		return types.Bool(reflect.DeepEqual(o.Raw, other.Raw))
	}
	return types.Bool(false)
}

// Type returns the CEL type of the CEL value wrapper.
func (o *Object[T]) Type() ref.Type {
	return cel.ObjectType(fmt.Sprintf("%T", o.Raw), traits.ReceiverType)
}

// Value returns the CEL value wrapper.
func (o *Object[T]) Value() any {
	return o
}

// RegisterObject registers a CEL value wrapper for a Go value with the
// type adapter and type provider, which are provided by the caller when
// constructing a CEL environment.
func RegisterObject[T any](ta TypeAdapter, tp *TypeProvider, objt *Object[T], t *types.Type, fields map[string]*types.FieldType) {
	ta[reflect.TypeOf(objt.Raw)] = func(value any) ref.Val {
		return objt
	}

	RegisterType(tp, t)

	RegisterStructType(tp, t.TypeName(), fields)
}
