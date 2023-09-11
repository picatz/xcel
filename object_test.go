package xcel_test

import (
	"fmt"
	"testing"

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

	obj, typ := xcel.NewObject(ta, tp, person, "Person")

	xcel.RegisterObject(ta, tp, obj, typ, map[string]*types.FieldType{
		"name": {
			Type: types.StringType,
			IsSet: ref.FieldTester(func(target any) bool {
				x := target.(*xcel.Object[*Person])

				if x.Raw == nil || x.Raw.Name == "" {
					return false
				}

				return true
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

				if x.Raw == nil || x.Raw.Age < 0 {
					return false
				}

				return true
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
	Name     string
	Age      int
	Tags     []string
	Parent   *Example
	Pressure float64
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
	}

	obj, typ := xcel.NewObject(ta, tp, ex, "Example")

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

				obj, _ := xcel.NewObject(ta, tp, x.Raw.Parent, "Example")

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
	})

	env, err := cel.NewEnv(
		cel.Types(typ),
		cel.Variable("obj", typ),
		cel.CustomTypeAdapter(ta),
		cel.CustomTypeProvider(tp),
	)

	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

	ast, iss := env.Compile("obj.name == 'test' && obj.age > 0 && ('test' in obj.tags) && obj.parent.name == 'root' && obj.pressure > 1.0")
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
