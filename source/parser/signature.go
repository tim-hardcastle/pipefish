package parser

import (
	"github.com/tim-hardcastle/pipefish/source/values"
)

// Sigs where the type is represented as a TypeNode.

type NameTypeAstPair struct {
	VarName string
	VarType TypeNode
}

type AstSig []NameTypeAstPair

// Sigs where the type is represented as an AbstractType.

type NameAbstractTypePair struct {
	VarName string
	VarType values.AbstractType
}

func (m NameAbstractTypePair) IsBling() bool {
	return m.VarType.Equals(values.MakeAbstractType(values.BLING))
}

func (m NameAbstractTypePair) Matches(n NameAbstractTypePair) bool {
	if m.IsBling() && n.IsBling() {
		return n.VarName == m.VarName
	}
	return m.VarType.Equals(n.VarType)
}

type AbstractSig []NameAbstractTypePair

func (p AbstractSig) String() string {
	result := "("
	sep := ""
	for _, pair := range p {
		result = result + sep + pair.VarName + " " + pair.VarType.String()
		sep = ", "
	}
	return result + ")"
}

func (ns AstSig) String() (result string) {
	if ns == nil {
		return "nil sig ast"
	}
	for _, v := range ns {
		if result != "" {
			result = result + ", "
		}
		if v.VarType == nil {
			result = result + v.VarName + " " + "no type"
		} else {
			result = result + v.VarName + " " + v.VarType.String()
		}
	}
	result = "(" + result + ")"
	return
}

