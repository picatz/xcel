package xcel

import (
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var _ types.Provider = &TypeProvider{}

type TypeProvider struct {
	Idents           map[string]ref.Val
	Types            map[string]*types.Type
	Stucts           map[string]map[string]*types.FieldType
	StructFieldTypes map[string]map[string]*types.FieldType
}

func NewTypeProvider() *TypeProvider {
	return &TypeProvider{
		Idents:           map[string]ref.Val{},
		Types:            map[string]*types.Type{},
		Stucts:           map[string]map[string]*types.FieldType{},
		StructFieldTypes: map[string]map[string]*types.FieldType{},
	}
}

func (TypeProvider) EnumValue(enumName string) ref.Val {
	return types.NewErr("not implemented")
}

func (tp *TypeProvider) FindIdent(identName string) (ref.Val, bool) {
	if v, ok := tp.Idents[identName]; ok {
		return v, true
	}
	return nil, false
}

func (tp *TypeProvider) FindStructType(structType string) (*types.Type, bool) {
	if t, ok := tp.Types[structType]; ok {
		return t, true
	}
	return nil, false
}

func (tp *TypeProvider) FindStructFieldNames(structType string) ([]string, bool) {
	if structType == "Example" {
		return []string{
			"name",
		}, true
	}

	if t, ok := tp.Stucts[structType]; ok {
		var names []string
		for name := range t {
			names = append(names, name)
		}
		return names, true
	}
	return nil, false
}

func (tp *TypeProvider) FindStructFieldType(messageType, fieldName string) (*types.FieldType, bool) {
	if t, ok := tp.StructFieldTypes[messageType]; ok {
		if ft, ok := t[fieldName]; ok {
			return ft, true
		}
	}
	return nil, false
}

func (TypeProvider) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	return types.NewErr("not implemented")
}

var DefaultTypeProvider = &TypeProvider{
	Idents:           map[string]ref.Val{},
	Types:            map[string]*types.Type{},
	Stucts:           map[string]map[string]*types.FieldType{},
	StructFieldTypes: map[string]map[string]*types.FieldType{},
}

func RegisterIdent(tp *TypeProvider, name string, value ref.Val) {
	tp.Idents[name] = value
}

func RegisterType(tp *TypeProvider, t *types.Type) {
	tp.Types[t.TypeName()] = t
}

func RegisterStructType(tp *TypeProvider, name string, fields map[string]*types.FieldType) {
	tp.Stucts[name] = fields
	registerStructFieldType(tp, name, fields)
}

func registerStructFieldType(tp *TypeProvider, name string, fields map[string]*types.FieldType) {
	tp.StructFieldTypes[name] = fields
}
