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

obj, typ := xcel.NewObject(person)

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

### Object Fields via Reflection

Fields can be registered via reflection, which is a bit more concise, but less flexible and less performant:

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

obj, typ := xcel.NewObject(person)

xcel.RegisterObject(ta, tp, obj, typ, xcel.NewFields(obj))
```

#### Benchmarks

Showing some minimal performance differences between manual fields and reflection based fields for the same object:

```go
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
```

<!-- This is a CEL expression, not JavaScript. But, it is close enough. -->

```javascript
obj.name == 'test' && obj.age > 0 && ('test' in obj.tags) && obj.parent.name == 'root' && obj.pressure > 1.0 && obj.fn(1) == '~1~' && has(obj.blob)
```

```console
$ go test -benchmem -run=^$ -bench ^BenchmarkNewObject github.com/picatz/xcel -v
goos: darwin
goarch: arm64
pkg: github.com/picatz/xcel
BenchmarkNewObjectManualFields
BenchmarkNewObjectManualFields-8          677050              1620 ns/op             784 B/op         22 allocs/op
BenchmarkNewObjectReflectionFields
BenchmarkNewObjectReflectionFields-8      546022              2138 ns/op             880 B/op         31 allocs/op
```
