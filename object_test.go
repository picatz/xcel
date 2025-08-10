package xcel_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/picatz/xcel"
)

func ExampleNewObject() {
	type Person struct {
		Name string
		Age  int
	}

	person := &Person{
		Name: "test",
		Age:  -1,
	}

	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	obj, typ := xcel.NewObject(person)

	xcel.RegisterObject(ta, tp, obj, typ, map[string]*types.FieldType{
		"name": {
			Type: types.StringType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Person])

				return x.Raw != nil && x.Raw.Name != ""
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Person])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Name, nil
			}),
		},
		"age": {
			Type: types.IntType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Person])

				return x.Raw != nil && x.Raw.Age >= 0
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Person])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Age, nil
			}),
		},
	})

	env, _ := cel.NewEnv(
		cel.Types(typ),
		cel.Variable("obj", typ),
		cel.CustomTypeAdapter(ta),
		cel.CustomTypeProvider(tp),
	)

	ast, _ := env.Compile("obj.name == 'test' && obj.age > 0")

	prg, _ := env.Program(ast)

	out, _, _ := prg.Eval(map[string]any{
		"obj": obj,
	})

	fmt.Println(out.Value())
	// Output: false
}

type Example struct {
	Nested
	NamedNested Nested
	Name        string
	Age         int
	Tags        []string
	Parent      *Example
	Pressure    float64
	Fn          func(int) string
	Blob        []byte
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	ExpiresAt   time.Time
}

type Nested struct {
	Toto string
}

func TestNewObject(t *testing.T) {
	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	ex := &Example{
		Name: "test",
		Age:  1,
		Tags: []string{"test"},
		Parent: &Example{
			Name: "root",
			Age:  -1,
			Tags: []string{"a", "b", "c"},
		},
		Pressure: 1.5,
		Fn: func(i int) string {
			return fmt.Sprintf("~%d~", i)
		},
	}
	ex.CreatedAt = time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	ua := time.Date(2025, 8, 1, 13, 0, 0, 0, time.UTC)
	ex.UpdatedAt = &ua
	ex.ExpiresAt = time.Date(2025, 8, 2, 12, 0, 0, 0, time.UTC)

	obj, typ := xcel.NewObject(ex)

	xcel.RegisterObject(ta, tp, obj, typ, map[string]*types.FieldType{
		"name": {
			Type: types.StringType,
			IsSet: ref.FieldTester(func(target any) bool {
				objW := target.(*xcel.Object[*Example])

				if objW.Raw == nil || objW.Raw.Name == "" {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				objW := target.(*xcel.Object[*Example])

				if objW.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return objW.Raw.Name, nil
			}),
		},
		"age": {
			Type: types.IntType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Age < 0 {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Age, nil
			}),
		},
		"tags": {
			Type: types.NewListType(types.StringType),
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || len(x.Raw.Tags) == 0 {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return types.NewStringList(ta, x.Raw.Tags), nil
			}),
		},
		"parent": {
			Type: typ,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Parent == nil {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				obj, _ := xcel.NewObject(x.Raw.Parent)

				return obj, nil
			}),
		},
		"pressure": {
			Type: types.DoubleType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Pressure <= 0 {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Pressure, nil
			}),
		},
		"created_at": {
			Type: types.TimestampType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])
				return x.Raw != nil && !x.Raw.CreatedAt.IsZero()
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])
				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}
				return types.Timestamp{Time: x.Raw.CreatedAt}, nil
			}),
		},
		"updated_at": {
			Type: types.TimestampType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])
				return x.Raw != nil && x.Raw.UpdatedAt != nil && !x.Raw.UpdatedAt.IsZero()
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])
				if x.Raw == nil || x.Raw.UpdatedAt == nil {
					return nil, fmt.Errorf("celval: object or updated_at is nil")
				}
				return types.Timestamp{Time: *x.Raw.UpdatedAt}, nil
			}),
		},
		"expires_at": {
			Type: types.TimestampType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])
				return x.Raw != nil && !x.Raw.ExpiresAt.IsZero()
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])
				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}
				return types.Timestamp{Time: x.Raw.ExpiresAt}, nil
			}),
		},
	})

	env, err := cel.NewEnv(
		cel.Types(typ),
		cel.Variable("obj", typ),
		cel.CustomTypeAdapter(ta),
		cel.CustomTypeProvider(tp),
		cel.Function("fn",
			cel.MemberOverload(
				"Example_int",
				[]*cel.Type{typ, cel.IntType},
				cel.StringType,
				cel.BinaryBinding(func(arg1, arg2 ref.Val) ref.Val {
					x := arg1.(*xcel.Object[*Example])
					y := arg2.(types.Int)

					return types.String(x.Raw.Fn(int(y)))
				}),
			),
		),
	)

	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

	ast, iss := env.Compile("obj.name == 'test' && obj.age > 0 && ('test' in obj.tags) && obj.parent.name == 'root' && obj.pressure > 1.0 && obj.fn(1) == '~1~' && has(obj.created_at) && has(obj.updated_at) && obj.updated_at > obj.created_at && obj.expires_at > obj.created_at")
	if iss.Err() != nil {
		t.Fatalf("failed to compile CEL expression: %v", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		t.Fatalf("failed to create CEL program: %v", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"obj": obj,
	})
	if err != nil {
		t.Fatalf("failed to evaluate program: %v", err)
	}

	if fmt.Sprintf("%v", out.Value()) != "true" {
		t.Fatalf("expected 'true' but got '%v'", out.Value())
	}
}

func TestNewObjectAndFields(t *testing.T) {
	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	ex := &Example{
		Name: "test",
		Age:  1,
		Tags: []string{"test"},
		Parent: &Example{
			Name: "root",
			Age:  -1,
			Tags: []string{"a", "b", "c"},
		},
		Pressure: 1.5,
		Fn: func(i int) string {
			return fmt.Sprintf("~%d~", i)
		},
		Blob: []byte("test"),
	}
	ex.CreatedAt = time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	ua := time.Date(2025, 8, 1, 13, 0, 0, 0, time.UTC)
	ex.UpdatedAt = &ua
	ex.ExpiresAt = time.Date(2025, 8, 2, 12, 0, 0, 0, time.UTC)

	obj, typ := xcel.NewObject(ex)

	xcel.RegisterObject(ta, tp, obj, typ, xcel.NewFields(obj))

	env, err := cel.NewEnv(
		cel.Types(typ),
		cel.Variable("obj", typ),
		cel.CustomTypeAdapter(ta),
		cel.CustomTypeProvider(tp),
		cel.Function("fn",
			cel.MemberOverload(
				"Example_int",
				[]*cel.Type{typ, cel.IntType},
				cel.StringType,
				cel.BinaryBinding(func(arg1, arg2 ref.Val) ref.Val {
					x := arg1.(*xcel.Object[*Example])
					y := arg2.(types.Int)

					return types.String(x.Raw.Fn(int(y)))
				}),
			),
		),
	)

	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

	ast, iss := env.Compile("obj.name == 'test' && obj.age > 0 && ('test' in obj.tags) && obj.parent.name == 'root' && obj.pressure > 1.0 && obj.fn(1) == '~1~' && has(obj.blob) && has(obj.created_at) && has(obj.updated_at) && obj.updated_at > obj.created_at && obj.expires_at > obj.created_at")
	if iss.Err() != nil {
		t.Fatalf("failed to compile CEL expression: %v", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		t.Fatalf("failed to create CEL program: %v", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"obj": obj,
	})
	if err != nil {
		t.Fatalf("failed to evaluate program: %v", err)
	}

	if fmt.Sprintf("%v", out.Value()) != "true" {
		t.Fatalf("expected 'true' but got '%v'", out.Value())
	}
}

func TestNewObjectNestedFields(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		checkValue func(t *testing.T, v any)
	}{
		{
			name: "access tested field",
			expr: "obj.toto == 'toto'",
			checkValue: func(t *testing.T, out any) {
				if fmt.Sprintf("%v", out) != "true" {
					t.Errorf("expected 'true' but got '%v'", out)
				}
			},
		},
		{
			name: "access named nested field",
			expr: "obj.named_nested.toto == 'titi'",
			checkValue: func(t *testing.T, out any) {
				if fmt.Sprintf("%v", out) != "true" {
					t.Errorf("expected 'true' but got '%v'", out)
				}
			},
		},
		{
			name: "timestamp presence and ordering",
			expr: "has(obj.created_at) && has(obj.updated_at) && obj.updated_at > obj.created_at",
			checkValue: func(t *testing.T, out any) {
				if fmt.Sprintf("%v", out) != "true" {
					t.Errorf("expected 'true' but got '%v'", out)
				}
			},
		},
	}

	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	ex := &Example{
		Nested: Nested{
			Toto: "toto",
		},
		NamedNested: Nested{
			Toto: "titi",
		},
		Name: "test",
		Age:  1,
	}
	ex.CreatedAt = time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	ua := time.Date(2025, 8, 1, 13, 0, 0, 0, time.UTC)
	ex.UpdatedAt = &ua
	ex.ExpiresAt = time.Date(2025, 8, 2, 12, 0, 0, 0, time.UTC)

	obj, typ := xcel.NewObject(ex)

	xcel.RegisterObject(ta, tp, obj, typ, xcel.NewFields(obj))

	env, err := cel.NewEnv(
		cel.Types(typ),
		cel.Variable("obj", typ),
		cel.CustomTypeAdapter(ta),
		cel.CustomTypeProvider(tp),
	)

	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

	for _, test := range tests {
		ast, iss := env.Compile(test.expr)
		if iss.Err() != nil {
			t.Fatalf("failed to compile CEL expression: %v", iss.Err())
		}

		prg, err := env.Program(ast)
		if err != nil {
			t.Fatalf("failed to create CEL program: %v", err)
		}

		out, _, err := prg.Eval(map[string]interface{}{
			"obj": obj,
		})
		if err != nil {
			t.Fatalf("failed to evaluate program: %v", err)
		}

		test.checkValue(t, out.Value())
	}
}

func BenchmarkNewObjectManualFields(b *testing.B) {
	// Benchmark the NewObject function with manually defined fields.
	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	ex := &Example{
		Name: "test",
		Age:  1,
		Tags: []string{"test"},
		Parent: &Example{
			Name: "root",
			Age:  -1,
			Tags: []string{"a", "b", "c"},
		},
		Pressure: 1.5,
		Fn: func(i int) string {
			return fmt.Sprintf("~%d~", i)
		},
		Blob: []byte("test"),
	}

	obj, typ := xcel.NewObject(ex)

	xcel.RegisterObject(ta, tp, obj, typ, map[string]*types.FieldType{
		"name": {
			Type: types.StringType,
			IsSet: ref.FieldTester(func(target any) bool {
				objW := target.(*xcel.Object[*Example])

				if objW.Raw == nil || objW.Raw.Name == "" {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				objW := target.(*xcel.Object[*Example])

				if objW.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return objW.Raw.Name, nil
			}),
		},
		"age": {
			Type: types.IntType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Age < 0 {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Age, nil
			}),
		},
		"tags": {
			Type: types.NewListType(types.StringType),
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || len(x.Raw.Tags) == 0 {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return types.NewStringList(ta, x.Raw.Tags), nil
			}),
		},
		"parent": {
			Type: typ,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Parent == nil {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				obj, _ := xcel.NewObject(x.Raw.Parent)

				return obj, nil
			}),
		},
		"pressure": {
			Type: types.DoubleType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Pressure <= 0 {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Pressure, nil
			}),
		},
		"blob": {
			Type: types.BytesType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || len(x.Raw.Blob) == 0 {
					return false
				}

				return true
			}),
			GetFrom: ref.FieldGetter(func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return types.Bytes(x.Raw.Blob), nil
			}),
		},
	})

	env, err := cel.NewEnv(
		cel.Types(typ),
		cel.Variable("obj", typ),
		cel.CustomTypeAdapter(ta),
		cel.CustomTypeProvider(tp),
		cel.Function("fn",
			cel.MemberOverload(
				"Example_int",
				[]*cel.Type{typ, cel.IntType},
				cel.StringType,
				cel.BinaryBinding(func(arg1, arg2 ref.Val) ref.Val {
					x := arg1.(*xcel.Object[*Example])
					y := arg2.(types.Int)

					return types.String(x.Raw.Fn(int(y)))
				}),
			),
		),
	)
	if err != nil {
		b.Fatalf("failed to create CEL environment: %v", err)
	}

	ast, iss := env.Compile("obj.name == 'test' && obj.age > 0 && ('test' in obj.tags) && obj.parent.name == 'root' && obj.pressure > 1.0 && obj.fn(1) == '~1~' && has(obj.blob)")
	if iss.Err() != nil {
		b.Fatalf("failed to compile CEL expression: %v", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		b.Fatalf("failed to create CEL program: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		out, _, err := prg.Eval(map[string]interface{}{
			"obj": obj,
		})
		if err != nil {
			b.Fatalf("failed to evaluate program: %v", err)
		}

		if fmt.Sprintf("%v", out.Value()) != "true" {
			b.Fatalf("expected 'true' but got '%v'", out.Value())
		}
	}

	b.StopTimer()
}

func BenchmarkNewObjectReflectionFields(b *testing.B) {
	// Benchmark the NewObject function with manually defined fields.
	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	ex := &Example{
		Name: "test",
		Age:  1,
		Tags: []string{"test"},
		Parent: &Example{
			Name: "root",
			Age:  -1,
			Tags: []string{"a", "b", "c"},
		},
		Pressure: 1.5,
		Fn: func(i int) string {
			return fmt.Sprintf("~%d~", i)
		},
		Blob: []byte("test"),
	}

	obj, typ := xcel.NewObject(ex)

	xcel.RegisterObject(ta, tp, obj, typ, xcel.NewFields(obj))

	env, err := cel.NewEnv(
		cel.Types(typ),
		cel.Variable("obj", typ),
		cel.CustomTypeAdapter(ta),
		cel.CustomTypeProvider(tp),
		cel.Function("fn",
			cel.MemberOverload(
				"Example_int",
				[]*cel.Type{typ, cel.IntType},
				cel.StringType,
				cel.BinaryBinding(func(arg1, arg2 ref.Val) ref.Val {
					x := arg1.(*xcel.Object[*Example])
					y := arg2.(types.Int)

					return types.String(x.Raw.Fn(int(y)))
				}),
			),
		),
	)
	if err != nil {
		b.Fatalf("failed to create CEL environment: %v", err)
	}

	ast, iss := env.Compile("obj.name == 'test' && obj.age > 0 && ('test' in obj.tags) && obj.parent.name == 'root' && obj.pressure > 1.0 && obj.fn(1) == '~1~' && has(obj.blob)")
	if iss.Err() != nil {
		b.Fatalf("failed to compile CEL expression: %v", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		b.Fatalf("failed to create CEL program: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		out, _, err := prg.Eval(map[string]interface{}{
			"obj": obj,
		})
		if err != nil {
			b.Fatalf("failed to evaluate program: %v", err)
		}

		if fmt.Sprintf("%v", out.Value()) != "true" {
			b.Fatalf("expected 'true' but got '%v'", out.Value())
		}
	}

	b.StopTimer()
}
