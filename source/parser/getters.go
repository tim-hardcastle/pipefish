package parser

import (
	"reflect"

	"github.com/tim-hardcastle/pipefish/source/dtypes"
	"github.com/tim-hardcastle/pipefish/source/token"
	"github.com/tim-hardcastle/pipefish/source/values"
)

// Auxiliary functions that extract data from data.
func (p *Parser) CanParse(tok token.Token, pos IdentifierPosition) (bool, *Parser) {	
	resolvingParser := p.getParserFromNamespace(tok)
	if resolvingParser == nil {
		return false, nil
	}
	_, ok := resolvingParser.BlingTree[BlingData{tok.Literal, pos}]
	return ok, resolvingParser
}

func (p *Parser) canBling(identifier string, pos IdentifierPosition) bool {
	return p.Common.BlingManager.canBling(identifier, pos)
}

func (p *Parser) didBling(identifier string, pos IdentifierPosition) bool {
	return p.Common.BlingManager.didBling(identifier, pos)
}

func (p *Parser) getSigFromArgs(args []Node, dflt TypeNode) (AstSig, bool) {
	sig := AstSig{}
	for _, arg := range args {
		partialSig, ok := p.ReparseSig(arg, dflt)
		if !ok {
			return nil, false
		}
		sig = append(sig, partialSig...)
	}
	return sig, true
}

func (p *Parser) GetVariablesFromSig(node Node) []string {
	result := []string{}
	sig, ok := p.ReparseSig(node, DUMMY_TYPE_AST)
	if !ok {
		return nil
	}
	for _, pair := range sig {
		result = append(result, pair.VarName)
	}
	return result
}

func (p *Parser) GetVariablesFromAstSig(sig AstSig) []string {
	result := []string{}
	for _, pair := range sig {
		result = append(result, pair.VarName)
	}
	return result
}


func (p *Parser) RecursivelySlurpReturnTypes(node Node) AstSig {
	if node == nil {
		return AstSig{}
	}
	switch typednode := node.(type) {
	case *InfixExpression:
		switch {
		case typednode.Token.Type == token.COMMA:
			LHS := p.RecursivelySlurpReturnTypes(typednode.Args[0])
			RHS := p.RecursivelySlurpReturnTypes(typednode.Args[2])
			return append(LHS, RHS...)
		default:
			p.Throw("parse/ret/a", typednode.GetToken())
		}
	case *TypeExpression:
		if typednode.TypeArgs == nil {
			return AstSig{NameTypeAstPair{VarName: "", VarType: &TypeWithName{typednode.Token, typednode.Operator}}}
		}
		return AstSig{NameTypeAstPair{VarName: "", VarType: typednode}}
	case *SuffixExpression:
		if typednode.Operator == "?" || typednode.Operator == "!" {
			return AstSig{NameTypeAstPair{VarName: "", VarType: &TypeSuffix{typednode.Token, typednode.Operator, p.RecursivelySlurpReturnTypes(typednode.Args[0])[0].VarType}}}
		}
	default:
		println("node is", typednode.String(), reflect.TypeOf(typednode).String())
		p.Throw("parse/ret/b", typednode.GetToken())
	}
	return nil
}

// Converts type expressions to TypeNodes, i.e. the sort of description of a type
// that we should be able to find in a function signature.
func (p *Parser) ToAstType(te *TypeExpression) TypeNode {
	if len(te.TypeArgs) == 0 {
		return &TypeWithName{Token: te.Token, OperatorName: te.Operator}
	}
	// This is either a bool, float, int, rune, string, type or enum literal, in which
	// case the whole thing should be, OR it's a type with parameters, or it's not well-
	// formed and shouldn't be here at all.
	indexArg := te.TypeArgs[0]
	if p.findTypeArgument(indexArg).T != values.ERROR {
		return p.toTypeWithArguments(te)
	}
	return p.toTypeWithParameters(te)
}

func (p *Parser) toTypeWithArguments(te *TypeExpression) *TypeWithArguments {
	result := TypeWithArguments{te.Token, te.Operator, []*Argument{}}
	for _, arg := range te.TypeArgs {
		v := p.findTypeArgument(arg)
		if v.T == values.ERROR {
			return &result
		}
		result.Arguments = append(result.Arguments, &Argument{*arg.GetToken(), v.T, v.V})
	}
	return &result
}

func (p *Parser) toTypeWithParameters(te *TypeExpression) *TypeWithParameters {
	sig, ok := p.getSigFromArgs(te.TypeArgs, &TypeWithName{OperatorName: "error"})
	if !ok {
		return nil
	}
	params := []*Parameter{}
	for _, pair := range sig {
		newParameter := &Parameter{pair.VarName, pair.VarType.String()}
		params = append(params, newParameter)
	}
	return &TypeWithParameters{te.Token, te.Operator, params}
}

func (p *Parser) findTypeArgument(arg Node) values.Value {
	switch arg := arg.(type) {
	case *Identifier:
		if p.IsEnumElement(arg.Value) {
			return values.Value{0, arg.Value} // We don't know the enum types yet so we kludge them in later.
		}
	case *BooleanLiteral:
		return values.Value{values.BOOL, arg.Value}
	case *FloatLiteral:
		return values.Value{values.FLOAT, arg.Value}
	case *IntegerLiteral:
		return values.Value{values.INT, arg.Value}
	case *RuneLiteral:
		return values.Value{values.RUNE, arg.Value}
	case *StringLiteral:
		return values.Value{values.STRING, arg.Value}
	case *TypeExpression:
		return values.Value{values.TYPE, p.ToAstType(arg)}
	}
	return values.Value{values.ERROR, nil}
}

func (p *Parser) IsEnumElement(name string) bool {
	_, ok := p.EnumElementNames[name]
	return ok
}

// Finds whether an identifier is in the right place to be a function, or whether it's being used
// as though it's a variable or constant.
func (p *Parser) isPositionallyFunctional() bool {
	if assignmentTokens.Contains(p.PeekToken.Type) {
		return false
	}
	if p.Common.BlingManager.canBling(p.PeekToken.Literal, ANY_BLING...) {
		return false
	}
	if p.PeekToken.Type == token.RPAREN || p.PeekToken.Type == token.PIPE ||
		p.PeekToken.Type == token.MAPPING || p.PeekToken.Type == token.FILTER ||
		p.PeekToken.Type == token.COLON || p.PeekToken.Type == token.MAGIC_COLON ||
		p.PeekToken.Type == token.COMMA || p.PeekToken.Type == token.RBRACK ||
		p.PeekToken.Type == token.RBRACE || p.PeekToken.Type == token.AND ||
		p.PeekToken.Type == token.OR {
		return false
	}
	if p.CurToken.Literal == "type" && p.IsTypePrefix(p.PeekToken.Literal) {
		return true
	}
	if p.Functions.Contains(p.CurToken.Literal) && p.Typenames.Contains(p.CurToken.Literal) {
		return p.typeIsFunctional()
	}

	if p.Functions.Contains(p.CurToken.Literal) && p.PeekToken.Type != token.EOF {
		return true
	}
	if literalsAndLParen.Contains(p.PeekToken.Type) {
		return true
	}
	if p.PeekToken.Type != token.IDENT {
		return false
	}
	if ok, _ := p.CanParse(p.PeekToken, INFIX); ok {
		return false
	}
	if ok, _ := p.CanParse(p.PeekToken, SUFFIX); ok {
		return false
	}
	return true
}

var (
	nativeInfixes = dtypes.MakeFromSlice([]token.TokenType{
		token.COMMA, token.EQ, token.NOT_EQ, token.ASSIGN, token.GVN_ASSIGN, token.FOR,
		token.GIVEN, token.LBRACK, token.MAGIC_COLON, token.MAGIC_SEMICOLON, token.PIPE, token.MAPPING,
		token.FILTER, token.IFLOG})
	lazyInfixes = dtypes.MakeFromSlice([]token.TokenType{token.AND,
		token.OR, token.COLON, token.SEMICOLON, token.NEWLINE})
)

// TODO --- there may at this point not be any need to have this different from any other function.
func (p *Parser) typeIsFunctional() bool {
	if p.Common.BlingManager.canBling(p.PeekToken.Literal, ANY_BLING...) {
		return false
	}
	if p.PeekToken.Type == token.RPAREN || p.PeekToken.Type == token.PIPE ||
		p.PeekToken.Type == token.MAPPING || p.PeekToken.Type == token.FILTER ||
		p.PeekToken.Type == token.COLON || p.PeekToken.Type == token.MAGIC_COLON ||
		p.PeekToken.Type == token.COMMA || p.PeekToken.Type == token.RBRACK ||
		p.PeekToken.Type == token.RBRACE || p.PeekToken.Literal == "?"  ||
		p.PeekToken.Type == token.AND || p.PeekToken.Type == token.OR {
		return false
	}
	if p.PeekToken.Type == token.EMDASH || p.PeekToken.Type == token.LBRACK {
		return true
	}
	if literalsAndLParen.Contains(p.PeekToken.Type) {
		return true
	}
	if p.PeekToken.Literal == "from" {
		return true
	}
	if ok, _ := p.CanParse(p.PeekToken, INFIX); ok {
		return false
	}
	if nativeInfixes.Contains(p.PeekToken.Type) {
		return false
	}
	if p.canBling(p.PeekToken.Literal, MIDFIX) {
		return false
	}
	if p.Functions.Contains(p.PeekToken.Literal) && p.PeekToken.Type != token.EOF {
		return true
	}
	if ok, _ := p.CanParse(p.PeekToken, PREFIX); ok {
		return p.PeekToken.Type != token.EOF
	}
	return p.PeekToken.Type != token.EOF
}

// Sometimes, as on the lhs of an assignment, something may be a signature but we don't
// know it until we hit the equals sign, at which point we have a node and we don't even know
// if it's a well-formed signature. We certainly haven't parsed it as one.
//
// So now we do that.
func (p *Parser) ReparseSig(node Node, dflt TypeNode) (AstSig, bool) {
	if node == nil {
		return AstSig{}, true
	}
	switch node := node.(type) {
	case *Identifier:
		return AstSig{NameTypeAstPair{node.Token.Literal, dflt}}, true
	case *InfixExpression:
		if node.Token.Type == token.COMMA {
			if len(node.GetArgs()) != 3 {
				p.Throw("comp/assign/lhs/a", node.GetToken())
			}
			left, ok := p.ReparseSig(node.GetArgs()[0], dflt)
			if !ok {
				return nil, false
			}
			right, ok := p.ReparseSig(node.GetArgs()[2], dflt)
			if !ok {
				return nil, false
			}
			for i := left.Len() - 1; i >= 0 && left[i].VarType == dflt; i-- {
				left[i].VarType = right[0].VarType
			} 
			return append(left, right...), true
		}
	case *PrefixExpression:
		result := AstSig{NameTypeAstPair{node.Token.Literal, dflt}}
		for _, arg := range node.Args {
			reparsedArg, ok := p.ReparseSig(arg, dflt)
			if !ok {
				return nil, false 
			}
			if reparsedArg[0].VarName == "" {
				result[result.Len()-1].VarType = reparsedArg[0].VarType
				reparsedArg = reparsedArg[1:]
			}
			result = append(result, reparsedArg...)
		}	
		return result, true
	case *TypeExpression:
		return AstSig{NameTypeAstPair{"", p.ToAstType(node)}}, true
	case *SuffixExpression:
		if len(node.Args) == 1 && node.Operator == "?" || node.Operator == "!" {
			left, ok := p.ReparseSig(node.Args[0], dflt)
			if !ok {
				return nil, false
			}
			left[0].VarType = &TypeSuffix{node.Token, node.Operator, left[0].VarType}
			return left, true
		}
	}
	p.Throw("parse/sig/c", node.GetToken())
	return nil, false
}
