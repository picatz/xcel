package xcel

import (
	"fmt"
	"reflect"
	"strings"
	"time"

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
func NewObject[T any](val T) (*Object[T], *types.Type) {
	return &Object[T]{Raw: val}, cel.ObjectType(reflect.TypeOf(val).String(), traits.ReceiverType)
}

// ConvertToNative converts the CEL value wrapper to a native Go value.
func (o *Object[T]) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if typeDesc == reflect.TypeOf(o.Raw) {
		return o.Raw, nil
	}
	return nil, fmt.Errorf("xcel: type conversion error from '%s' to '%s'", o.Type(), typeDesc)
}

// ConvertToType converts the CEL value wrapper to a CEL value of the specified type.
func (o *Object[T]) ConvertToType(typeValue ref.Type) ref.Val {
	if typeValue == o.Type() {
		return o
	}
	return types.NewErr("xcel: type conversion error from '%s' to '%s'", o.Type(), typeValue)
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

// NewFields returns a map[string]*types.FieldType for the given object type
// wrapping a Go struct pointer value.
func NewFields[T any](objt *Object[T]) map[string]*types.FieldType {
	fields := map[string]*types.FieldType{}

	// Get the struct from the pointer.
	v := reflect.ValueOf(objt.Raw).Elem()

	// Use a recursive helper function to handle nested structs
	processFields[T](fields, v, "")

	return fields
}

// processFields recursively processes fields of a struct, including nested ones.
// It uses a prefix to build the full path for nested fields (e.g., "Address.City").
func processFields[T any](fields map[string]*types.FieldType, v reflect.Value, prefix string) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)
		name := fieldType.Name

		// Check if the field is a struct (and not a time.Time, which is a common struct but should be treated as a primitive).
		if field.Kind() == reflect.Struct && fieldType.Type != reflect.TypeOf(time.Time{}) {
			// If it's a nested struct, recurse.
			newPrefix := name
			if prefix != "" {
				newPrefix = prefix + "." + name
			}
			processFields[T](fields, field, newPrefix)
			continue
		}

		// Build the full field name (e.g., "Address.City").
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}

		// Get the field value for CEL type conversion.
		value := field.Interface()

		var celType *types.Type

		// Convert the field value to a CEL value, if possible, default to object.
		switch value.(type) {
		case string:
			celType = types.StringType
		case int:
			celType = types.IntType
		case float64:
			celType = types.DoubleType
		case bool:
			celType = types.BoolType
		case []string:
			celType = types.NewListType(types.StringType)
		default:
			celType = cel.ObjectType(reflect.TypeOf(value).String(), traits.ReceiverType)
		}

		// Use lower case for the field name.
		fields[strings.ToLower(name)] = &types.FieldType{
			Type: celType,
			IsSet: func(target any) bool {
				// Navigate to the correct field using the full path.
				x := reflect.ValueOf(target.(*Object[T]).Raw).Elem()
				f := getNestedField(x, fullName)

				if !f.IsValid() {
					return false
				}
				switch f.Kind() {
				case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan, reflect.Interface:
					return !f.IsNil()
				default:
					return true
				}
			},
			GetFrom: func(target any) (any, error) {
				// Navigate to the correct field using the full path.
				x := target.(*Object[T]).Raw

				v2 := reflect.ValueOf(x).Elem()

				// Get index of the field.
				f := getNestedField(v2, fullName)

				if !f.IsValid() {
					return nil, fmt.Errorf("field %s not found", fullName)
				}

				// Get the field value.
				value2 := f.Interface()

				vt, ok := value2.(T)
				if !ok {
					return value2, nil
				}

				// Create a CEL object from the field value.
				obj, _ := NewObject(vt)

				return obj, nil
			},
		}
	}
}

// getNestedField navigates a struct path to find a nested field.
func getNestedField(v reflect.Value, path string) reflect.Value {
	parts := strings.Split(path, ".")
	for _, part := range parts {
		v = v.FieldByName(part)
		if !v.IsValid() {
			return reflect.Value{}
		}
	}
	return v
}
