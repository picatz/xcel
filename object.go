package xcel

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

var (
	goTimeType = reflect.TypeOf(time.Time{})
)

// presenceIsSet reports whether fv should be considered present for CEL has().
// Rules:
//   - time.Time: zero value is not present; non-zero is present.
//   - *time.Time: nil is not present; non-nil is present only if non-zero.
//   - Pointers, slices, maps, interfaces, funcs, chans: present iff non-nil.
//   - All other kinds: present (even if the zero value).
func presenceIsSet(fv reflect.Value, _ reflect.StructField) bool {
	// time.Time
	if fv.Type() == goTimeType {
		return !fv.IsZero()
	}
	// *time.Time
	if fv.Kind() == reflect.Ptr && fv.Type().Elem() == goTimeType {
		if fv.IsNil() {
			return false
		}
		return !fv.Elem().IsZero()
	}
	switch fv.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func, reflect.Chan:
		return !fv.IsNil()
	default:
		return true
	}
}

// typeNameOf returns the Go type name for a given reflect.Type.
func typeNameOf(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.PkgPath() == "" {
		return t.String()
	}
	return t.PkgPath() + "." + t.Name()
}

// wrapperTypeName returns the Go type name used for the CEL wrapper type *Object[T].
func wrapperTypeName[T any]() string {
	return fmt.Sprintf("%T", (*Object[T])(nil))
}

// celTypeForField returns the CEL type corresponding to the declared Go field type.
// Special cases:
//   - time.Time and *time.Time → cel.TimestampType
//   - []byte → cel.BytesType
//   - []string → cel.List(String)
//
// Primitive scalars map to their obvious CEL types. All other types are exposed as
// object types so that member dispatch uses the wrapper.
func celTypeForField(sf reflect.StructField) *types.Type {
	t := sf.Type
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == goTimeType {
		return types.TimestampType
	}
	switch t.Kind() {
	case reflect.String:
		return types.StringType
	case reflect.Int, reflect.Int32, reflect.Int64:
		return types.IntType
	case reflect.Uint, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return types.UintType
	case reflect.Float32, reflect.Float64:
		return types.DoubleType
	case reflect.Bool:
		return types.BoolType
	case reflect.Slice:
		et := t.Elem()
		if et.Kind() == reflect.Uint8 {
			return types.BytesType
		} // []byte
		if et.Kind() == reflect.String {
			return types.NewListType(types.StringType)
		} // []string
	}
	return cel.ObjectType(typeNameOf(sf.Type), traits.ReceiverType)
}

// fieldNameFor returns the snake_case CEL field name for a Go struct field.

// Object wraps a Go value for use as a CEL object. The wrapper type is used as the
// CEL object type so member functions dispatch to the wrapper.
type Object[T any] struct {
	Raw T
}

// NewObject returns a CEL wrapper for val and its CEL object type.
func NewObject[T any](val T) (*Object[T], *types.Type) {
	// Use the wrapper type as the CEL object type so member dispatch passes the
	// wrapper (matching tests which assert *Object[T]).
	return &Object[T]{Raw: val}, cel.ObjectType(wrapperTypeName[T](), traits.ReceiverType)
}

// ConvertToNative returns the underlying Go value when typeDesc matches the wrapped type.
func (o *Object[T]) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if typeDesc == reflect.TypeOf(o.Raw) {
		return o.Raw, nil
	}
	return nil, fmt.Errorf("xcel: type conversion error from '%s' to '%s'", o.Type(), typeDesc)
}

// ConvertToType implements ref.Val.ConvertToType for the wrapper.
func (o *Object[T]) ConvertToType(typeValue ref.Type) ref.Val {
	if typeValue == o.Type() {
		return o
	}
	return types.NewErr("xcel: type conversion error from '%s' to '%s'", o.Type(), typeValue)
}

// Equal reports whether other is an *Object[T] with an equal underlying value.
func (o *Object[T]) Equal(other ref.Val) ref.Val {
	if other, ok := other.(*Object[T]); ok {
		return types.Bool(reflect.DeepEqual(o.Raw, other.Raw))
	}
	return types.Bool(false)
}

// Type returns the CEL type of the wrapper.
func (o *Object[T]) Type() ref.Type {
	return cel.ObjectType(wrapperTypeName[T](), traits.ReceiverType)
}

// Value returns the wrapper itself. Adapters handle unwrapping when needed.
func (o *Object[T]) Value() any {
	return o
}

// RegisterObject registers objt and its type with the given adapter and provider.
// It derives field metadata from reflection (optionally overlaid by fields),
// registers the struct type, and registers reachable named nested struct types so
// nested field access type-checks at compile time.
func RegisterObject[T any](ta TypeAdapter, tp *TypeProvider, objt *Object[T], t *types.Type, fields map[string]*types.FieldType) {
	ta[reflect.TypeOf(objt.Raw)] = func(value any) ref.Val {
		// Return the registered wrapper to guarantee the exact wrapper type
		// is used at call sites (important for member overload assertions in tests).
		return objt
	}

	// Build from reflection first, then overlay any provided entries so callers
	// can override behavior for specific fields (e.g., presence predicates).
	auto := NewFields(objt)
	if fields == nil {
		fields = auto
	} else {
		for k, v := range fields {
			auto[k] = v
		}
		fields = auto
	}

	ta[reflect.TypeOf(objt)] = func(value any) ref.Val {
		if rv, ok := value.(ref.Val); ok {
			return rv
		}
		wrapped, _ := NewObject(value.(T))
		return wrapped
	}

	RegisterType(tp, t)
	RegisterStructType(tp, t.TypeName(), fields)
	registerNestedTypes(tp, objt.Raw, map[reflect.Type]struct{}{})
}

// registerNestedTypes registers named nested struct types reachable from raw so that
// nested field access can be type-checked. It follows pointers and recurses into
// nested structs while avoiding cycles via visited.
func registerNestedTypes(tp *TypeProvider, raw any, visited map[reflect.Type]struct{}) {
	v := reflect.ValueOf(raw)
	vt := v.Type()
	for vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}
	if _, seen := visited[vt]; seen {
		return
	}
	visited[vt] = struct{}{}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i)
		if !ft.IsExported() {
			continue
		}
		fieldValue := v.Field(i)

		// Treat struct or pointer-to-struct (excluding time.Time) as a named nested struct.
		underlying := ft.Type
		if underlying.Kind() == reflect.Ptr {
			underlying = underlying.Elem()
		}
		isStructLike := (fieldValue.Kind() == reflect.Struct) || (fieldValue.Kind() == reflect.Ptr && fieldValue.Elem().Kind() == reflect.Struct)
		if isStructLike && underlying != goTimeType && !ft.Anonymous {
			// Build a pointer to the nested struct for consistent typing.
			ptr := fieldValue
			if fieldValue.Kind() != reflect.Ptr {
				if fieldValue.CanAddr() {
					ptr = fieldValue.Addr()
				} else {
					ptr = reflect.New(underlying)
				}
			}

			// Create a temporary object to compute its (wrapper) type and fields.
			obj, nestedType := NewObject(ptr.Interface())
			RegisterType(tp, nestedType)
			RegisterStructType(tp, nestedType.TypeName(), newFields(obj))

			// Recurse into the nested struct.
			registerNestedTypes(tp, ptr.Interface(), visited)
		}
	}
}

// NewFields returns CEL field metadata for the immediate fields of objt.
func NewFields[T any](objt *Object[T]) map[string]*types.FieldType {
	return newFields(objt)
}

func newFields[T any](objt *Object[T]) map[string]*types.FieldType {
	fields := map[string]*types.FieldType{}
	v := reflect.ValueOf(objt.Raw)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	processImmediateFields[T](fields, v)
	return fields
}

// toSnakeCase converts an exported Go field name to snake_case.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				if prev != '_' && (unicode.IsLower(prev) || (i+1 < len(s) && unicode.IsLower(rune(s[i+1])))) {
					b.WriteRune('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// processImmediateFields records field metadata for v's immediate fields.
// Anonymous embedded struct fields have their leaf fields promoted at this level.
// Named struct fields are exposed as nested objects; their inner fields are
// provided by separate nested type registration.
func processImmediateFields[T any](fields map[string]*types.FieldType, v reflect.Value) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	rootType := v.Type()
	for i := 0; i < rootType.NumField(); i++ {
		ft := rootType.Field(i)
		if !ft.IsExported() { // ignore unexported
			continue
		}
		fieldValue := v.Field(i)

		// Skip function fields; they are not exposed as CEL fields and may conflict
		// with registered member overloads of the same name.
		if fieldValue.Kind() == reflect.Func {
			continue
		}

		// Handle interface fields
		if ft.Type.Kind() == reflect.Interface {
			fieldValue = fieldValue.Elem() // dereference interface to get the concrete type
		}

		// Handle struct or pointer-to-struct fields specially (except time.Time which should
		// behave like a primitive value).
		underlying := ft.Type
		if underlying.Kind() == reflect.Ptr {
			underlying = underlying.Elem()
		}
		isStructLike := (fieldValue.Kind() == reflect.Struct) || (fieldValue.Kind() == reflect.Ptr && fieldValue.Elem().Kind() == reflect.Struct)
		if isStructLike && underlying != goTimeType {
			// Promote embedded fields: make leaf fields available at this level
			processPromotedFields[T](fields, fieldValue, ft.Name)
			continue
		}

		// Primitive / non-struct field at this level.
		fullPath := ft.Name
		name := toSnakeCase(strings.ReplaceAll(fullPath, ".", "_"))

		sf := ft // capture for closure
		if _, exists := fields[name]; exists {
			panic(fmt.Sprintf("xcel: field name collision for CEL name '%s' on %s (Go field: %s)", name, rootType, sf.Name))
		}
		celTy := celTypeForField(sf)
		fields[name] = &types.FieldType{
			Type: celTy,
			IsSet: func(target any) bool {
				x := reflect.ValueOf(target.(*Object[T]).Raw)
				if x.Kind() == reflect.Ptr {
					x = x.Elem()
				}
				f := getNestedField(x, fullPath)
				if !f.IsValid() {
					return false
				}
				return presenceIsSet(f, sf)
			},
			GetFrom: func(target any) (any, error) {
				x := reflect.ValueOf(target.(*Object[T]).Raw)
				if x.Kind() == reflect.Ptr {
					x = x.Elem()
				}
				f := getNestedField(x, fullPath)
				if !f.IsValid() {
					return nil, fmt.Errorf("field %s not found", fullPath)
				}
				if v, ok := normalizeForCEL(f); ok {
					return v, nil
				}
				return f.Interface(), nil
			},
		}
	}
}

// processPromotedFields promotes leaf fields from an anonymous embedded struct so
// they are visible on the parent object while retaining reflection access via prefix.
func processPromotedFields[T any](fields map[string]*types.FieldType, v reflect.Value, prefix string) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i)
		if !ft.IsExported() {
			continue
		}
		fieldValue := v.Field(i)

		// Build the reflection path like "Nested.Field".
		fullPath := prefix + "." + ft.Name
		name := toSnakeCase(strings.ReplaceAll(fullPath, ".", "_"))

		// Only register leaf / non-structs here; named nested structs should be
		// reached through their parent field (which is not anonymous).
		if fieldValue.Kind() == reflect.Struct && ft.Type != goTimeType {
			// Recurse further for deeply embeddings.
			processPromotedFields[T](fields, fieldValue, prefix+"."+ft.Name)
			continue
		}

		// Skip function fields; they are not exposed as CEL fields and may conflict
		// with registered member overloads of the same name.
		if fieldValue.Kind() == reflect.Func {
			continue
		}

		sf := ft // capture for closure and diagnostics
		if _, exists := fields[name]; exists {
			panic(fmt.Sprintf("xcel: field name collision for CEL name '%s' on %s (Go field: %s)", name, v.Type(), sf.Name))
		}
		celTy := celTypeForField(sf)
		fields[name] = &types.FieldType{
			Type: celTy,
			IsSet: func(target any) bool {
				x := reflect.ValueOf(target.(*Object[T]).Raw)
				if x.Kind() == reflect.Ptr {
					x = x.Elem()
				}
				f := getNestedField(x, fullPath)
				if !f.IsValid() {
					return false
				}
				return presenceIsSet(f, sf)
			},
			GetFrom: func(target any) (any, error) {
				x := reflect.ValueOf(target.(*Object[T]).Raw)
				if x.Kind() == reflect.Ptr {
					x = x.Elem()
				}
				f := getNestedField(x, fullPath)
				if !f.IsValid() {
					return nil, fmt.Errorf("field %s not found", fullPath)
				}
				if v, ok := normalizeForCEL(f); ok {
					return v, nil
				}
				return f.Interface(), nil
			},
		}
	}
}

// getNestedField returns the value at path (e.g., "Parent.Child.Field") within v,
// following pointers as needed. It returns an invalid reflect.Value if the path
// cannot be resolved to a struct field.
func getNestedField(v reflect.Value, path string) reflect.Value {
	if v.Kind() == reflect.Interface {
		v = v.Elem() // dereference interface to get the concrete type
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for _, part := range strings.Split(path, ".") {
		if v.Kind() == reflect.Interface {
			v = v.Elem() // dereference interface to get the concrete type
		}
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		f := v.FieldByName(part)
		if !f.IsValid() {
			return reflect.Value{}
		}
		v = f
	}
	return v
}

// normalizeForCEL converts supported native values to their CEL equivalents.
// Currently: time.Time and *time.Time → cel.Timestamp.
func normalizeForCEL(fv reflect.Value) (any, bool) {
	// time.Time
	if fv.Type() == goTimeType {
		return types.Timestamp{Time: fv.Interface().(time.Time)}, true
	}
	// *time.Time
	if fv.Kind() == reflect.Ptr && fv.Elem().IsValid() && fv.Elem().Type() == goTimeType {
		return types.Timestamp{Time: fv.Elem().Interface().(time.Time)}, true
	}
	return nil, false
}
