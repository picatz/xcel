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
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Person])

				return x.Raw != nil && x.Raw.Name != ""
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Person])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Name, nil
			},
		},
		"age": {
			Type: types.IntType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Person])

				return x.Raw != nil && x.Raw.Age >= 0
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Person])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Age, nil
			},
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
	Toto    string
	Level   int64
	Enabled bool
}

// K8sEvent is a minimal local replica of the external interface used only for
// validating interface field handling.
//
// https://github.com/kubescape/node-agent/blob/7f1258f0de66ecedf8d7c0c2cd3e2d280d99d929/pkg/utils/events.go#L12-L16
type K8sEvent interface {
	GetPod() string
	GetNamespace() string
}

// TestRuntime corresponds to BasicRuntimeMetadata (subset needed).
type TestRuntime struct {
	ContainerID string
}

// TestK8s corresponds to BasicK8sMetadata (subset needed).
type TestK8s struct {
	ContainerName string
	Namespace     string
}

// TestCommonData groups runtime and k8s metadata; in the original chain this
// data was reachable through anonymous embeddings culminating in an anonymous
// CommonData, so we mimic by embedding this struct anonymously further down.
type TestCommonData struct {
	Runtime TestRuntime
	K8s     TestK8s
}

// TestBase anonymously embeds TestCommonData so its leaf struct fields (Runtime,
// K8s) are promoted upward through subsequent anonymous embeddings per the
// reflection logic in xcel.
type TestBase struct {
	TestCommonData
}

// TestTraceEvent adds process-related data and anonymously embeds TestBase.
type TestTraceEvent struct {
	TestBase
	Pid     uint64
	ExePath string
	Args    []string
}

func (t *TestTraceEvent) GetPod() string       { return t.TestCommonData.K8s.ContainerName }
func (t *TestTraceEvent) GetNamespace() string { return t.TestCommonData.K8s.Namespace }

// TestExecEvent anonymously embeds TestTraceEvent, continuing promotion.
type TestExecEvent struct {
	TestTraceEvent
}

// TestEnrichedEvent mirrors the external EnrichedEvent shape we rely on only
// for the interface field `Event` whose dynamic underlying concrete type has
// the anonymous embedding chain described above.
//
// https://github.com/kubescape/node-agent/blob/7f1258f0de66ecedf8d7c0c2cd3e2d280d99d929/pkg/ebpf/events/enriched_event.go#L21-L29
type TestEnrichedEvent struct {
	Event K8sEvent // interface field to exercise interface + struct handling
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
			IsSet: func(target any) bool {
				objW := target.(*xcel.Object[*Example])

				if objW.Raw == nil || objW.Raw.Name == "" {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				objW := target.(*xcel.Object[*Example])

				if objW.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return objW.Raw.Name, nil
			},
		},
		"age": {
			Type: types.IntType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Age < 0 {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Age, nil
			},
		},
		"tags": {
			Type: types.NewListType(types.StringType),
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || len(x.Raw.Tags) == 0 {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return types.NewStringList(ta, x.Raw.Tags), nil
			},
		},
		"parent": {
			Type: typ,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Parent == nil {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				obj, _ := xcel.NewObject(x.Raw.Parent)

				return obj, nil
			},
		},
		"pressure": {
			Type: types.DoubleType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Pressure <= 0 {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Pressure, nil
			},
		},
		"created_at": {
			Type: types.TimestampType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])
				return x.Raw != nil && !x.Raw.CreatedAt.IsZero()
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])
				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}
				return types.Timestamp{Time: x.Raw.CreatedAt}, nil
			},
		},
		"updated_at": {
			Type: types.TimestampType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])
				return x.Raw != nil && x.Raw.UpdatedAt != nil && !x.Raw.UpdatedAt.IsZero()
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])
				if x.Raw == nil || x.Raw.UpdatedAt == nil {
					return nil, fmt.Errorf("celval: object or updated_at is nil")
				}
				return types.Timestamp{Time: *x.Raw.UpdatedAt}, nil
			},
		},
		"expires_at": {
			Type: types.TimestampType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])
				return x.Raw != nil && !x.Raw.ExpiresAt.IsZero()
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])
				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}
				return types.Timestamp{Time: x.Raw.ExpiresAt}, nil
			},
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
			name: "access embedded nested field",
			expr: "obj.nested.toto == 'toto'",
			checkValue: func(t *testing.T, out any) {
				if fmt.Sprintf("%v", out) != "true" {
					t.Errorf("expected 'true' but got '%v'", out)
				}
			},
		},
		{
			name: "promoted int field",
			expr: "obj.level == 7",
			checkValue: func(t *testing.T, out any) {
				if fmt.Sprintf("%v", out) != "true" {
					t.Errorf("expected 'true' but got '%v'", out)
				}
			},
		},
		{
			name: "promoted int literal == field",
			expr: "7 == obj.level",
			checkValue: func(t *testing.T, out any) {
				if fmt.Sprintf("%v", out) != "true" {
					t.Errorf("expected 'true' but got '%v'", out)
				}
			},
		},
		{
			name: "promoted bool field",
			expr: "obj.enabled == true",
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
			Toto:    "toto",
			Level:   7,
			Enabled: true,
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

	fmt.Println(ex.Toto)             // Through nested indirection
	fmt.Println(ex.Nested.Toto)      // Through nested struct
	fmt.Println(ex.NamedNested.Toto) // Through named nested struct

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
		t.Run(test.name, func(t *testing.T) {
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
		})
	}
}

func TestNewObjectWithEvent(t *testing.T) {
	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	ex := &TestEnrichedEvent{
		Event: &TestExecEvent{
			TestTraceEvent: TestTraceEvent{
				TestBase: TestBase{
					TestCommonData: TestCommonData{
						K8s:     TestK8s{ContainerName: "test"},
						Runtime: TestRuntime{ContainerID: "test"},
					},
				},
				Pid:     1234,
				ExePath: "/usr/bin/test-process",
				Args:    []string{"test-process", "arg1"},
			},
		},
	}

	// fmt.Println(ex.Event.(*TestExecEvent).Runtime.ContainerID)

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

	ast, iss := env.Compile("obj.event.runtime.container_id == 'test'")
	if iss.Err() != nil {
		t.Fatalf("failed to compile CEL expression: %v", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		t.Fatalf("failed to create CEL program: %v", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{"obj": obj})
	if err != nil {
		t.Fatalf("failed to evaluate program: %v", err)
	}
	if fmt.Sprintf("%v", out.Value()) != "true" {
		t.Fatalf("expected 'true' but got '%v'", out.Value())
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
			IsSet: func(target any) bool {
				objW := target.(*xcel.Object[*Example])

				if objW.Raw == nil || objW.Raw.Name == "" {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				objW := target.(*xcel.Object[*Example])

				if objW.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return objW.Raw.Name, nil
			},
		},
		"age": {
			Type: types.IntType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Age < 0 {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Age, nil
			},
		},
		"tags": {
			Type: types.NewListType(types.StringType),
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || len(x.Raw.Tags) == 0 {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return types.NewStringList(ta, x.Raw.Tags), nil
			},
		},
		"parent": {
			Type: typ,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Parent == nil {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				obj, _ := xcel.NewObject(x.Raw.Parent)

				return obj, nil
			},
		},
		"pressure": {
			Type: types.DoubleType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || x.Raw.Pressure <= 0 {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return x.Raw.Pressure, nil
			},
		},
		"blob": {
			Type: types.BytesType,
			IsSet: func(target any) bool {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil || len(x.Raw.Blob) == 0 {
					return false
				}

				return true
			},
			GetFrom: func(target any) (any, error) {
				x := target.(*xcel.Object[*Example])

				if x.Raw == nil {
					return nil, fmt.Errorf("celval: object is nil")
				}

				return types.Bytes(x.Raw.Blob), nil
			},
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

// AllPrimitives exercises every primitive kind handled by wrapForCEL.
type AllPrimitives struct {
	Name   string
	Age    int64
	Count  uint64
	Score  float64
	Active bool
}

// TestPreWrappedEquality verifies that field == literal AND literal == field
// both evaluate correctly when GetFrom returns pre-wrapped cel-go types.
func TestPreWrappedEquality(t *testing.T) {
	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	ap := &AllPrimitives{
		Name:   "dns",
		Age:    42,
		Count:  100,
		Score:  3.14,
		Active: true,
	}

	obj, typ := xcel.NewObject(ap)
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

	tests := []struct {
		name string
		expr string
	}{
		{"field == literal string", `obj.name == "dns"`},
		{"literal == field string", `"dns" == obj.name`},
		{"field == literal int", `obj.age == 42`},
		{"literal == field int", `42 == obj.age`},
		{"field == literal uint", `obj.count == 100u`},
		{"literal == field uint", `100u == obj.count`},
		{"field == literal double", `obj.score == 3.14`},
		{"literal == field double", `3.14 == obj.score`},
		{"field == literal bool", `obj.active == true`},
		{"literal == field bool", `true == obj.active`},
		{"string contains", `obj.name.contains("dn")`},
		{"string startsWith", `obj.name.startsWith("d")`},
		{"string endsWith", `obj.name.endsWith("ns")`},
		{"string matches", `obj.name.matches("^d.s$")`},
		{"int comparison", `obj.age > 40`},
		{"uint comparison", `obj.count >= 100u`},
		{"double comparison", `obj.score < 4.0`},
		{"bool negation", `obj.active != false`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ast, iss := env.Compile(tc.expr)
			if iss.Err() != nil {
				t.Fatalf("compile: %v", iss.Err())
			}
			prg, err := env.Program(ast)
			if err != nil {
				t.Fatalf("program: %v", err)
			}
			out, _, err := prg.Eval(map[string]any{"obj": obj})
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if out.Value() != true {
				t.Fatalf("expected true, got %v", out.Value())
			}
		})
	}
}

// TestPreWrappedTypeAdapterPassthrough verifies that NativeToValue passes
// through pre-wrapped ref.Val values without re-conversion.
func TestPreWrappedTypeAdapterPassthrough(t *testing.T) {
	ta := xcel.NewTypeAdapter()

	cases := []struct {
		name string
		val  ref.Val
	}{
		{"String", types.String("hello")},
		{"Int", types.Int(42)},
		{"Uint", types.Uint(100)},
		{"Double", types.Double(3.14)},
		{"Bool", types.Bool(true)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ta.NativeToValue(tc.val)
			if result != tc.val {
				t.Fatalf("expected pass-through, got different value: %v vs %v", result, tc.val)
			}
		})
	}
}

// BenchmarkEvalWithPrewrap benchmarks field access with pre-wrapped values
// (reflection-based NewFields) to measure alloc reduction.
func BenchmarkEvalWithPrewrap(b *testing.B) {
	ta, tp := xcel.NewTypeAdapter(), xcel.NewTypeProvider()

	ap := &AllPrimitives{
		Name:   "dns",
		Age:    42,
		Count:  100,
		Score:  3.14,
		Active: true,
	}

	obj, typ := xcel.NewObject(ap)
	xcel.RegisterObject(ta, tp, obj, typ, xcel.NewFields(obj))

	env, err := cel.NewEnv(
		cel.Types(typ),
		cel.Variable("obj", typ),
		cel.CustomTypeAdapter(ta),
		cel.CustomTypeProvider(tp),
	)
	if err != nil {
		b.Fatalf("failed to create CEL environment: %v", err)
	}

	ast, iss := env.Compile(`obj.name == "dns" && obj.age == 42 && obj.count == 100u && obj.score > 3.0 && obj.active == true`)
	if iss.Err() != nil {
		b.Fatalf("compile: %v", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		b.Fatalf("program: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		out, _, err := prg.Eval(map[string]any{"obj": obj})
		if err != nil {
			b.Fatalf("eval: %v", err)
		}
		if out.Value() != true {
			b.Fatalf("expected true, got %v", out.Value())
		}
	}
}
