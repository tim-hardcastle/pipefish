package vm

import (
	"github.com/tim-hardcastle/pipefish/source/token"
	"github.com/tim-hardcastle/pipefish/source/values"
)

// Struct and functions for handling the concrete type information.

type supertype int

const (
	NATIVE supertype = iota
	ENUM
	STRUCT
)

type TypeInformation interface {
	GetName(flavor descriptionFlavor) string
	getPath() string
	IsEnum() bool
	IsStruct() bool
	isSnippet() bool
	IsClone() bool
	IsGoType() bool
	IsPrivate() bool
	IsMandatoryImport() bool
	IsClonedBy() values.AbstractType
}

type WrapperType struct {
	Name    string
	Path    string
	Private bool
	Gotype  string
}

func (t WrapperType) GetName(flavor descriptionFlavor) string {
	if flavor == LITERAL {
		return t.Path + t.Name
	}
	return string(t.Name)
}

func (t WrapperType) IsEnum() bool {
	return false
}

func (t WrapperType) IsStruct() bool {
	return false
}

func (t WrapperType) isSnippet() bool {
	return false
}

func (t WrapperType) IsPrivate() bool {
	return t.Private
}

func (t WrapperType) IsClone() bool {
	return false
}

func (t WrapperType) IsGoType() bool {
	return true
}

func (t WrapperType) getPath() string {
	return t.Path
}

func (t WrapperType) IsMandatoryImport() bool {
	return false
}

func (WrapperType) IsClonedBy() values.AbstractType {
	return values.MakeAbstractType()
}

type BuiltinType struct {
	name   string
	clones values.AbstractType
}

func (t BuiltinType) GetName(flavor descriptionFlavor) string {
	return string(t.name)
}

func (t BuiltinType) IsEnum() bool {
	return false
}

func (t BuiltinType) IsStruct() bool {
	return false
}

func (t BuiltinType) isSnippet() bool {
	return false
}

func (t BuiltinType) IsClone() bool {
	return false
}

func (t BuiltinType) IsGoType() bool {
	return false
}

func (t BuiltinType) IsPrivate() bool {
	return false
}

func (t BuiltinType) getPath() string {
	return ""
}

func (t BuiltinType) IsMandatoryImport() bool {
	return true
}

func (t BuiltinType) IsClonedBy() values.AbstractType {
	return t.clones
}

func (t BuiltinType) AddClone(v values.ValueType) BuiltinType {
	t.clones = t.clones.Insert(v)
	return t
}

type EnumType struct {
	Name          string
	Path          string
	ElementNames  []string
	ElementValues values.Value // A list.
	Private       bool
	IsMI          bool
}

func (t EnumType) GetName(flavor descriptionFlavor) string {
	if flavor == LITERAL {
		return t.Path + t.Name
	}
	return t.Name
}

func (t EnumType) IsEnum() bool {
	return true
}

func (t EnumType) IsStruct() bool {
	return false
}

func (t EnumType) isSnippet() bool {
	return false
}

func (t EnumType) IsPrivate() bool {
	return t.Private
}

func (t EnumType) IsClone() bool {
	return false
}

func (t EnumType) IsGoType() bool {
	return false
}

func (t EnumType) getPath() string {
	return t.Path
}

func (t EnumType) IsMandatoryImport() bool {
	return t.IsMI
}

func (EnumType) IsClonedBy() values.AbstractType {
	return values.MakeAbstractType()
}

// Contains the information necessary to perform the runtime checks on type constructors
// on structs and clones.
type TypeCheck struct {
	CallAddress  uint32 // The address we `jsr`` to to perform the typecheck.
	InLoc        uint32 // The location of the first argument of the constructor.
	ResultLoc    uint32 // Where we put the error/ok.
	TokNumberLoc uint32 // Contains a location which contains an integer which is an index of the tokens in the vm.
}

type CloneType struct {
	Name          string
	Path          string
	Parent        values.ValueType
	Private       bool
	IsSliceable   bool
	IsFilterable  bool
	IsMappable    bool
	IsMI          bool
	Using         []token.Token // TODO --- this is used during API serialization only and can be stored somewhere else once we move that to initialization time.
	TypeCheck     *TypeCheck
	TypeArguments []values.Value
}

func (t CloneType) GetName(flavor descriptionFlavor) string {
	if flavor == LITERAL {
		return t.Path + t.Name
	}
	return t.Name
}

func (t CloneType) IsEnum() bool {
	return false
}

func (t CloneType) IsStruct() bool {
	return false
}

func (t CloneType) isSnippet() bool {
	return false
}

func (t CloneType) IsPrivate() bool {
	return t.Private
}

func (t CloneType) IsClone() bool {
	return true
}

func (t CloneType) IsGoType() bool {
	return false
}

func (t CloneType) getPath() string {
	return t.Path
}

func (t CloneType) IsMandatoryImport() bool {
	return t.IsMI
}

func (CloneType) IsClonedBy() values.AbstractType {
	return values.MakeAbstractType()
}

func (t CloneType) AddTypeCheck(tc *TypeCheck) CloneType {
	t.TypeCheck = tc
	return t
}

type StructType struct {
	Name                 string
	Path                 string
	LabelNumbers         []int
	LabelValues          values.Value // A list.
	Snippet              bool
	Private              bool
	AbstractStructFields []values.AbstractType
	ResolvingMap         map[int]int // TODO --- it would probably be better to implment this as a linear search below a given threshhold and a binary search above it.
	IsMI                 bool
	TypeCheck            *TypeCheck
	TypeArguments        []values.Value
}

func (t StructType) GetName(flavor descriptionFlavor) string {
	if flavor == LITERAL {
		return t.Path + t.Name
	}
	return t.Name
}

func (t StructType) IsEnum() bool {
	return false
}

func (t StructType) IsStruct() bool {
	return true
}

func (t StructType) isSnippet() bool {
	return t.Snippet
}

func (t StructType) IsPrivate() bool {
	return t.Private
}

func (t StructType) IsClone() bool {
	return false
}

func (t StructType) IsGoType() bool {
	return false
}

func (t StructType) Len() int {
	return len(t.LabelNumbers)
}

func (t StructType) getPath() string {
	return t.Path
}

func (t StructType) IsMandatoryImport() bool {
	return t.IsMI
}

func (StructType) IsClonedBy() values.AbstractType {
	return values.MakeAbstractType()
}

func (t StructType) AddLabels(labels []int) StructType {
	t.ResolvingMap = make(map[int]int)
	for k, v := range labels {
		t.ResolvingMap[v] = k
	}
	return t
}

func (t StructType) Resolve(labelNumber int) int {
	result, ok := t.ResolvingMap[labelNumber]
	if ok {
		return result
	}
	return -1
}

func (t StructType) AddTypeCheck(tc *TypeCheck) StructType {
	t.TypeCheck = tc
	return t
}
