package parser

import (
	"github.com/tim-hardcastle/pipefish/source/values"
)

// Sigs where the type is represented as a TypeNode.

type NameTypeAstPair struct {
	VarName string
	VarType TypeNode
}

func (ntp NameTypeAstPair) GetName() string {
	return ntp.VarName
}

func (ntp NameTypeAstPair) GetType() any {
	return ntp.VarType
}

type AstSig []NameTypeAstPair

// Sigs where the type is represented as an AbstractType.

type NameAbstractTypePair struct {
	VarName string
	VarType values.AbstractType
}

func (natp NameAbstractTypePair) GetName() string {
	return natp.VarName
}

func (natp NameAbstractTypePair) GetType() any {
	return natp.VarType
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

func (s AstSig) Len() int {
	return len(s)
}

func (s AstSig) GetVarType(i int) any {
	return s[i].GetType()
}

func (s AstSig) GetVarName(i int) string {
	return s[i].VarName
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

