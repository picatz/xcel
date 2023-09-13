# xcel

Extend [`CEL`](https://github.com/google/cel-spec) expressions with custom (native Go) objects and functions.

## Usage

```console
$ go get github.com/picatz/xcel@latest
```

## Example

```go
type Person struct {
	Name string
	Age  int
}

person := &Person{
	Name: "test",
	Age:  -1,
}

ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

obj, typ := xcel.NewObject(ta, tp, person)

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
```