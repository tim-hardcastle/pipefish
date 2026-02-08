package parser

import (
	"strconv"

	"github.com/tim-hardcastle/pipefish/source/dtypes"
	"github.com/tim-hardcastle/pipefish/source/token"
	"github.com/tim-hardcastle/pipefish/source/values"
)

// This parses descriptions of types (a) when they're defined (b) when they're mentioned
// in type signatures. In the bodies of functions, the arguments of parameterized types are
// just normal expressions.

// Things which are not type names but can be used for constructing types or for other
// purposes:
var PSEUDOTYPES = dtypes.MakeFromSlice([]string{"clone", "clones"})

type typePrecedence = int

const (
	T_LOWEST = iota
	T_OR
	T_AND
	T_SUFFIX
)

func (p *Parser) IsTypePrefix(s string) bool {
	return s == "..." || (p.Typenames.Contains(s) ||
		PSEUDOTYPES.Contains(s) || p.ParameterizedTypes.Contains(s))
}

func (p *Parser) ParseType(prec typePrecedence) TypeNode {
	rp := p.getParserFromNamespace(p.PeekToken)
	if !((p.PeekToken.Type == token.DOTDOTDOT) ||
		(p.PeekToken.Type == token.IDENT && rp.IsTypePrefix(p.PeekToken.Literal))) {
		p.Throw("parse/type/exists", &p.PeekToken)
		return nil
	}
	p.NextToken()
	return p.ParseTypeFromCurTok(prec)
}

func (p *Parser) ParseTypeFromCurTok(prec typePrecedence) TypeNode {
	var leftExp TypeNode
	tok := p.CurToken
	// Prefixes
	if p.PeekToken.Type == token.LBRACE {
		leftExp = p.parseParamsOrArgs()
		p.NextToken()
	} else {
		if p.CurToken.Type == token.DOTDOTDOT {
			right := p.ParseType(T_LOWEST)
			leftExp = &TypeDotDotDot{tok, right}
		} else {
			leftExp = &TypeWithName{tok, p.CurToken.Literal}
		}
	}
	// Infixes
	for prec <= p.peekTypePrecedence() && p.PeekToken.Type == token.IDENT &&
		(p.PeekToken.Literal == "/" || p.PeekToken.Literal == "&") {
		infixTok := p.PeekToken
		newPrec := p.peekTypePrecedence()
		p.NextToken()
		leftExp = &TypeInfix{infixTok, infixTok.Literal, leftExp, p.ParseType(newPrec)}
	}
	// Suffixes
	for p.PeekToken.Type == token.IDENT &&
		(p.PeekToken.Literal == "?" || p.PeekToken.Literal == "!") {
		p.NextToken()
		leftExp = &TypeSuffix{p.CurToken, p.CurToken.Literal, leftExp}
	}
	return leftExp
}

func (p *Parser) peekTypePrecedence() typePrecedence {
	switch p.PeekToken.Literal {
	case "/":
		return T_OR
	case "&":
		return T_AND
	case "?", "!":
		return T_SUFFIX
	default:
		return T_LOWEST
	}
}

func (p *Parser) parseParamsOrArgs() TypeNode {
	nameTok := p.CurToken
	p.NextToken() // The one with the name in.
	// So we're now at the token with the `{}`, which we won't skip over because sluriping
	// the type needs to be done with a peek first and a NextToken afterwards.
	if p.PeekToken.Type == token.IDENT && !(p.IsTypePrefix(p.PeekToken.Literal)) &&
		!(p.IsEnumElement(p.PeekToken.Literal)) {
		p.NextToken()
		result := p.parseParams(nameTok)
		return result
	}
	result := p.parseArgs(nameTok)
	return result
}

var acceptableTypes = dtypes.MakeFromSlice([]string{"float", "int", "string", "rune", "bool", "type"})

func (p *Parser) parseParams(nameTok token.Token) TypeNode {
	indexTok := p.CurToken
	blank := true
	result := TypeWithParameters{nameTok, nameTok.Literal, []*Parameter{}}
	for {
		tok := &p.CurToken
		if p.CurToken.Type != token.IDENT {
			p.Throw("parse/param/name", tok)
			break
		}
		result.Parameters = append(result.Parameters, &Parameter{p.CurToken.Literal, ""})
		blank = blank && p.CurToken.Literal == "_"
		p.NextToken()
		if p.CurToken.Type == token.IDENT {
			if acceptableTypes.Contains(p.CurToken.Literal) || p.EnumTypeNames.Contains(p.CurToken.Literal) {
				for _, v := range result.Parameters {
					if v.Type == "" {
						v.Type = p.CurToken.Literal
					}
				}
			} else {
				p.Throw("parse/param/type", tok)
			}
			p.NextToken()
		}
		if p.CurToken.Type == token.COMMA {
			p.NextToken()
			continue
		}
		if p.CurToken.Type == token.RBRACE {
			break
		}
		p.Throw("parse/param/form", tok)
		break
	}
	if blank {
		return &TypeWithName{indexTok, result.String()}
	}
	return &result
}

func (p *Parser) parseArgs(nameTok token.Token) TypeNode {
	result := TypeWithArguments{nameTok, nameTok.Literal, []*Argument{}}
	for {
		tok := p.PeekToken
		var newArg *Argument
		switch tok.Type {
		case token.FLOAT:
			number, _ := strconv.ParseFloat(tok.Literal, 64)
			newArg = &Argument{tok, values.FLOAT, number}
		case token.INT:
			number, _ := strconv.Atoi(tok.Literal)
			newArg = &Argument{tok, values.INT, number}
		case token.STRING:
			newArg = &Argument{tok, values.STRING, tok.Literal}
		case token.RUNE:
			newArg = &Argument{tok, values.RUNE, tok.Literal}
		case token.IDENT:
			if p.IsTypePrefix(tok.Literal) {
				newType := p.ParseType(T_LOWEST)
				newArg = &Argument{tok, values.TYPE, newType}
			} else {
				newArg = &Argument{tok, values.UNDEFINED_TYPE, tok.Literal} // This may or may not be an element of an enum and we're not going to sort that out in the parser.
			}
		case token.FALSE:
			newArg = &Argument{tok, values.BOOL, false}
		case token.TRUE:
			newArg = &Argument{tok, values.BOOL, true}
		default:
			p.Throw("parse/instance/value", &tok)
		}
		result.Arguments = append(result.Arguments, newArg)
		if tok.Type != token.IDENT || p.EnumElementNames.Contains(tok.Literal) { // In which case parsing the type will have moved us on to the next token.
			p.NextToken()
		}
		if p.PeekToken.Type == token.COMMA {
			p.NextToken()
			continue
		}
		if p.PeekToken.Type == token.RBRACE {
			break
		}
		p.Throw("parse/instance/form", &tok)
		break
	}
	return &result
}
