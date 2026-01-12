package parser

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/tim-hardcastle/pipefish/source/dtypes"
	"github.com/tim-hardcastle/pipefish/source/err"
	"github.com/tim-hardcastle/pipefish/source/lexer"
	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/text"
	"github.com/tim-hardcastle/pipefish/source/token"
	"github.com/tim-hardcastle/pipefish/source/values"
)

type Parser struct {

	// Temporary state: things that are used to parse one line.

	TokenizedCode lexer.TokenSupplier
	nesting       dtypes.Stack[token.Token]
	CurToken      token.Token
	PeekToken     token.Token
	Logging       bool

	// Things that need to be attached to every parser: common information about the type system, functions, etc.
	Common *CommonParserBindle

	// Permanent state: things set up by the initializer which are
	// then constant for the lifetime of the service.

	// Names/token types of identifiers.
	Functions          dtypes.Set[string]
	Forefixes          dtypes.Set[string]
	Midfixes           dtypes.Set[string]
	Endfixes           dtypes.Set[string]
	Typenames          dtypes.Set[string]
	EnumTypeNames      dtypes.Set[string]
	EnumElementNames   dtypes.Set[string]
	ParameterizedTypes dtypes.Set[string]

	ParTypes map[string]TypeExpressionInfo // Maps type operators to their numbers in the ParameterizedTypeInfo map in the VM.
	// Something of a kludge. We want instances of parameterized types to be made if they're mentioned in the code.
	// Since only the parser is in a position to notice this, we pile up such mentions in this list.
	// Since we only want the parser to do this during initialization, we have a guard saying that
	// we don't do this if the list is `nil`, and then the intializer sets the list to `nil` as soon
	// as it's used it, thus discarding the data.
	ParTypeInstances map[string]*TypeWithArguments

	ExternalParsers map[string]*Parser     // A map from the name of the external service to the parser of the service. This should be the same as the one in the vm.
	NamespaceBranch map[string]*ParserData // Map from the namespaces immediately available to this parser to the parsers they access.
	NamespacePath   string                 // The chain of namespaces that got us to this parser, as a string.
	Private         bool                   // Indicates if it's the parser of a private library/external/whatevs.

	BlingTree blingTree // Filled up by the `AddWordsToParser` method and then used by the bling manager in the Common Parser Bindle.

}

func New(common *CommonParserBindle, source, sourceCode, namespacePath string) *Parser {
	p := &Parser{
		Logging:            true,
		nesting:            *dtypes.NewStack[token.Token](),
		Functions:          make(dtypes.Set[string]),
		Forefixes:          make(dtypes.Set[string]),
		Midfixes:           make(dtypes.Set[string]),
		Endfixes:           make(dtypes.Set[string]),
		Typenames:          make(dtypes.Set[string]),
		EnumTypeNames:      make(dtypes.Set[string]),
		EnumElementNames:   make(dtypes.Set[string]),
		ParameterizedTypes: make(dtypes.Set[string]),
		ParTypeInstances:   map[string]*TypeWithArguments{},
		ParTypes:           make(map[string]TypeExpressionInfo),
		NamespaceBranch:    make(map[string]*ParserData),
		ExternalParsers:    make(map[string]*Parser),
		NamespacePath:      namespacePath,
		Common:             common,
		BlingTree:          newBlingTree(),
	}
	p.Common.Sources[source] = strings.Split(sourceCode, "\n") // TODO --- something else.
	p.TokenizedCode = lexer.NewRelexer(source, sourceCode)
	p.Typenames = p.Typenames.Add("any")
	p.Typenames = p.Typenames.Add("enum")
	p.Typenames = p.Typenames.Add("struct")

	p.Functions.Add("builtin")

	return p
}

type TypeExpressionInfo struct {
	VmTypeInfo          uint32
	IsClone             bool
	PossibleReturnTypes values.AbstractType
}

// Parses one line of code supplied as a string.
func (p *Parser) ParseLine(source, input string) Node {
	p.ResetAfterError()
	rl := lexer.NewRelexer(source, input)
	p.TokenizedCode = rl
	result := p.ParseTokenizedChunk()
	p.Common.Errors = append(rl.GetErrors(), p.Common.Errors...)
	return result
}

// Sets the parser up with the appropriate relexer and position to parse a string.
func (p *Parser) PrimeWithString(source, input string) {
	p.ResetParser()
	rl := lexer.NewRelexer(source, input)
	p.TokenizedCode = rl
	p.SafeNextToken()
	p.SafeNextToken()
}

// Sets the parser up with the appropriate relexer and position to parse a string.
func (p *Parser) PrimeWithTokenSupplier(source lexer.TokenSupplier) {
	if tcc, ok := source.(*TokenizedCodeChunk); ok {
		tcc.ToStart()
	}
	p.TokenizedCode = source
	p.SafeNextToken()
	p.SafeNextToken()
}

// Parses a type supplied as a string, for use in 'parser_test.go'.
func (p *Parser) ParseTypeFromString(input string) TypeNode {
	p.PrimeWithString("test", input)
	result := p.ParseTypeFromCurTok(T_LOWEST)
	p.Common.Errors = append(p.TokenizedCode.(*lexer.Relexer).GetErrors(), p.Common.Errors...)
	return result
}

// Some supporting types for the parser and their methods.

// For data that needs to be shared by all parsers. It is initialized when we start initializing a service
// and passed to the first parser, which then passes it down to its children.
type CommonParserBindle struct {
	InterfaceBacktracks []BkInterface
	Errors              []*err.Error
	IsBroken            bool
	Sources             map[string][]string
	// This helps keep track of the bling and --- TODO --- will eventually replace pretty much
	// everything else that handles bling.
	BlingManager *BlingManager
}

// Initializes the common parser bindle.
func NewCommonParserBindle() *CommonParserBindle {
	result := CommonParserBindle{
		Errors:              []*err.Error{},            // This is where all the errors emitted by enything end up.
		Sources:             make(map[string][]string), // Source code --- TODO: remove.
		InterfaceBacktracks: []BkInterface{},           // Although these are only ever used at compile time, they are emited by the `seekFunctionCall` method, which belongs to the compiler.
		BlingManager:        newBlingManager(),
	}
	return &result
}

// When we dispatch on a function which is semantically available to us because it fulfills an interface, but we
// haven't compiled it yet, this keeps track of where we backtrack to.
type BkInterface struct {
	Fn   any // This will in fact always be of type *compiler.CallInfo.
	Addr uint32
}

// Stores parse code chunks for subsequent tokenization.
type ParsedCodeChunks []Node

// Stores information about other parsers. TODO, deprecate.
type ParserData struct {
	Parser         *Parser
	ScriptFilepath string
}

func (p *Parser) ParseExpression(precedence int) Node {

	if literals.Contains(p.CurToken.Type) && literalsAndLParen.Contains(p.PeekToken.Type) {
		p.Throw("parse/before/a", &p.CurToken, &p.PeekToken)
	}
	var leftExp Node
	noNativePrefix := false
	switch p.CurToken.Type {

	// These just need a rhs.
	case token.EVAL, token.GLOBAL, token.XCALL:
		leftExp = p.parsePrefixExpression()

	// Remaining prefix-position token types are in alphabetical order.
	case token.BREAK:
		leftExp = p.parseBreak()
	case token.CONTINUE:
		leftExp = p.parseContinue()
	case token.ELSE:
		leftExp = p.parseElse()
	case token.EMDASH:
		leftExp = p.parseSnippetExpression(p.CurToken)
	case token.FALSE:
		leftExp = p.parseBooleanLiteral()
	case token.FLOAT:
		leftExp = p.parseFloatLiteral()
	case token.FOR:
		leftExp = p.parseForExpression()
	case token.GOLANG:
		leftExp = p.parseGolangExpression()
	case token.INT:
		leftExp = p.parseIntegerLiteral()
	case token.LBRACK:
		leftExp = p.parseListExpression()
	case token.LPAREN:
		leftExp = p.parseGroupedExpression()
	case token.NOT:
		leftExp = p.parseNativePrefixExpression()
	case token.PRELOG:
		leftExp = p.parsePrelogExpression()
	case token.STRING:
		leftExp = p.parseStringLiteral()
	case token.RANGE:
		leftExp = p.parseNativePrefixExpression()
	case token.RUNE:
		leftExp = p.parseRuneLiteral()
	case token.TRUE:
		leftExp = p.parseBooleanLiteral()
	case token.TRY:
		leftExp = p.parseTryExpression()
	case token.UNWRAP:
		leftExp = p.parseNativePrefixExpression()
	case token.VALID:
		leftExp = p.parseNativePrefixExpression()
	default:
		noNativePrefix = true
	}

	// So what we're going to do is find out if the identifier *thinks* it's a function, i.e. if it precedes
	// something that's a prefix (in the broader sense, i.e. an identifier, literal, LPAREN, etc). But not a
	// minus sign, that would be confusing, people can use parentheses.
	// If so, then we will parse it as though it's a Function, and it had better turn out to be a lambda at
	// runtime. If it isn't, then we'll treat it as an identifier.
	// TODO -- why is builtin not a native prefix?
	// 'from' isn't because we want to be able to use it as an infix and 'for' may end up the same way for the same reason.
	if noNativePrefix {
		if p.CurToken.Type == token.IDENT {
			if p.CurToken.Literal == "builtin" {
				p.CurToken.Type = token.BUILTIN
				leftExp = p.parseBuiltInExpression()
				return leftExp
			}
			// Here we step in and deal with things that are functions and values, like the type conversion
			// functions and their associated types. Before we look them up as functions, we want to
			// be sure that they're not in such a position that they're being used as literals.
			_, resolvingParser := p.CanParse(p.CurToken, PREFIX)
			if resolvingParser == nil {
				return nil
			}
			if resolvingParser.IsTypePrefix(p.CurToken.Literal) && !(p.CurToken.Literal == "func") { // TODO --- really it should nly happen for clones and structs.
				tok := p.CurToken
				operator := tok.Literal
				var typeArgs []Node
				if p.PeekToken.Type == token.LBRACE {
					p.NextToken()
					p.NextToken()
					typeArgsNode := p.ParseExpression(FPREFIX)
					typeArgs = p.RecursivelyListify(typeArgsNode)
					if p.PeekToken.Type == token.RBRACE {
						p.NextToken()
					} else {
						p.Throw("parse/rbrace", &p.CurToken)
					}
				}
				if p.typeIsFunctional() {
					p.NextToken()
					var right Node
					if p.CurToken.Type == token.LPAREN || p.CurToken.Type == token.LBRACK {
						right = p.ParseExpression(MINUS)
					} else {
						right = p.ParseExpression(FPREFIX)
					}
					args := p.RecursivelyListify(right)
					leftExp = &TypePrefixExpression{Token: tok, Operator: operator, Args: args, TypeArgs: typeArgs}
					if p.ParTypeInstances != nil { // We set this to nil after initialization so that we don't go on scraping things into it.
						astType := p.ToAstType(&TypeExpression{Token: tok, Operator: operator, TypeArgs: typeArgs})
						if astType, ok := astType.(*TypeWithArguments); ok {
							p.ParTypeInstances[astType.String()] = astType
						}
					}
				} else {
					leftExp = &TypeExpression{Token: tok, Operator: operator, TypeArgs: typeArgs}
					if p.ParTypeInstances != nil { // We set this to nil after initialization so that we don't go on scraping things into it.
						astType := p.ToAstType(leftExp.(*TypeExpression))
						if astType, ok := astType.(*TypeWithArguments); ok {
							p.ParTypeInstances[astType.String()] = astType
						}
					}
				}
			} else {
				ok, rp := p.CanParse(p.CurToken, UNFIX)
				rp.CurToken = p.CurToken
				rp.PeekToken = p.PeekToken
				if !resolvingParser.isPositionallyFunctional() {
					switch {
					case ok:
						leftExp = p.parseUnfixExpression()
					case p.Common.BlingManager.canBling(p.CurToken.Literal, ENDFIX):
						p.Common.BlingManager.doBling(p.CurToken.Literal, ENDFIX)
						leftExp = &Bling{Token: p.CurToken, Value: p.CurToken.Literal}
					default:
						leftExp = p.parseIdentifier()
					}
				} else {
					switch {
					case p.CurToken.Literal == "func":
						leftExp = p.parseLambdaExpression()
						return leftExp // TODO --- don't.
					case p.CurToken.Literal == "from":
						leftExp = p.parseFromExpression()
						return leftExp
					default:
						switch {
						case p.Common.BlingManager.canBling(p.CurToken.Literal, FOREFIX):
							p.Common.BlingManager.doBling(p.CurToken.Literal, FOREFIX)
							blingIs := &Bling{Token: p.CurToken, Value: p.CurToken.Literal}
							dummyCommaTok := p.CurToken
							dummyCommaTok.Literal = ","
							p.NextToken()
							restOfExpIs := p.ParseExpression(FPREFIX)
							leftExp = &InfixExpression{dummyCommaTok, ",", []Node{blingIs, &Bling{Value: ",", Token: dummyCommaTok}, restOfExpIs}}
						default:
							p.Common.BlingManager.startFunction(p.CurToken.Literal, PREFIX, resolvingParser.BlingTree)
							leftExp = p.parsePrefixExpression()
							p.Common.BlingManager.stopFunction()
						}
					}
				}
			}
		} else {
			p.Throw("parse/prefix", &p.CurToken)
			return nil
		}
	}

	if p.PeekToken.Type == token.EMDASH {
		right := p.parseSnippetExpression(p.PeekToken)
		tok := token.Token{token.COMMA, ",", p.PeekToken.Line, p.PeekToken.ChStart,
			p.PeekToken.ChEnd, p.PeekToken.Source, ""}
		children := []Node{leftExp, &Bling{tok, ","}, right}
		result := &InfixExpression{tok, ",", children}
		p.NextToken()
		return result
	}
	if p.Common.BlingManager.canEndfix(p.PeekToken.Literal) {
		p.NextToken()
		p.Common.BlingManager.doBling(p.CurToken.Literal, ANY_BLING...)
		blingIs := &Bling{Token: p.CurToken, Value: p.CurToken.Literal}
		dummyCommaTok := p.CurToken
		dummyCommaTok.Literal = ","
		leftExp = &InfixExpression{dummyCommaTok, ",", []Node{leftExp, &Bling{Value: ",", Token: dummyCommaTok}, blingIs}}
	}
	for p.Common.BlingManager.canBling(p.PeekToken.Literal, MIDFIX) {
		p.Common.BlingManager.doBling(p.PeekToken.Literal, MIDFIX)
		p.NextToken()
		leftExp = p.parseInfixExpression(leftExp)
	}
	for precedence < p.peekPrecedence() {
		// We look for suffixes.
		for {
			ok, rp := p.CanParse(p.PeekToken, SUFFIX)
			if rp == nil {
				p.Throw("parse/namespace", &p.CurToken, &p.PeekToken)
				return nil
			}
			if !(rp.IsTypePrefix(p.PeekToken.Literal) || ok || p.PeekToken.Type == token.DOTDOTDOT) {
				break
			}
			if p.CurToken.Type == token.NOT || p.CurToken.Type == token.IDENT && p.CurToken.Literal == "-" || p.CurToken.Type == token.ELSE {
				p.Throw("parse/before/b", &p.CurToken, &p.PeekToken)
				return nil
			}
			maybeType := p.PeekToken.Literal
			if rp.IsTypePrefix(maybeType) {
				tok := p.PeekToken
				typeAst := p.ParseType(T_LOWEST)
				// TODO --- the namespace needs to be represented in the type
				ty := typeAst
				if ty, ok := ty.(*TypeDotDotDot); ok && ty.Right == nil {
					leftExp = &SuffixExpression{
						Token:    p.CurToken,
						Operator: p.CurToken.Literal,
						Args:     p.RecursivelyListify(leftExp),
					}
				} else {
					leftExp = &TypeSuffixExpression{tok, typeAst, p.RecursivelyListify(leftExp)}
				}
			} else {
				p.NextToken()
				leftExp = p.parseSuffixExpression(leftExp)
			}
		}
		if p.PeekToken.Type == token.LOG {
			p.NextToken()
			leftExp = p.parseLogExpression(leftExp)
		}

		if precedence >= p.peekPrecedence() {
			break
		}
		// We move on to infixes.
		ok, rp := p.CanParse(p.PeekToken, INFIX)
		if rp == nil {
			p.Throw("parse/namespace/b", &p.PeekToken)
			return nil
		}
		foundInfix := nativeInfixes.Contains(p.PeekToken.Type) ||
			lazyInfixes.Contains(p.PeekToken.Type) ||
			ok
		// TODO --- find some way of eliciting this error or prove that this can't happen.
		if !foundInfix {
			p.Throw("parse/wut", &p.PeekToken)
			return nil
		}
		p.NextToken()
		if foundInfix {
			switch {
			case lazyInfixes.Contains(p.CurToken.Type):
				leftExp = p.parseLazyInfixExpression(leftExp)
			case p.CurToken.Type == token.LBRACK:
				leftExp = p.parseIndexExpression(leftExp)
			case p.CurToken.Type == token.PIPE || p.CurToken.Type == token.MAPPING ||
				p.CurToken.Type == token.FILTER:
				leftExp = p.parseStreamingExpression(leftExp)
			case p.CurToken.Type == token.IFLOG:
				leftExp = p.parseIfLogExpression(leftExp)
			case p.CurToken.Type == token.FOR:
				leftExp = p.parseForAsInfix(leftExp) // For the (usual) case where the 'for' is inside a 'from' and the leftExp is, or should be, the bound variables of the loop.
			case p.CurToken.Type == token.EQ || p.CurToken.Type == token.NOT_EQ:
				leftExp = p.parseComparisonExpression(leftExp)
			default:
				_, resolvingParser := p.CanParse(p.CurToken, INFIX)
				if resolvingParser == nil {
					return nil
				}
				p.Common.BlingManager.startFunction(p.CurToken.Literal, INFIX, resolvingParser.BlingTree)
				leftExp = p.parseInfixExpression(leftExp)
				p.Common.BlingManager.stopFunction()
			}
		}
	}
	if leftExp == nil {
		if p.CurToken.Type == token.EOF {
			p.Throw("parse/line", &p.CurToken)
			return nil
		}
		if p.CurToken.Literal == "<-|" || p.CurToken.Literal == ")" || // TODO --- it's not clear this or the following error can ever actually be thrown.
			p.CurToken.Literal == "]" || p.CurToken.Literal == "}" {
			p.Throw("parse/close", &p.CurToken)
			return nil
		}
		p.Throw("parse/missing", &p.CurToken)
		return nil
	}
	return leftExp
}

// Now we have all the functions with names of the form `parseXxxxx`, arranged in alphabetical order.

func (p *Parser) parseAssignmentExpression(left Node) Node {
	expression := &AssignmentExpression{
		Token: p.CurToken,
		Left:  left,
	}
	precedence := p.curPrecedence()
	p.NextToken()
	expression.Right = p.ParseExpression(precedence)
	return expression
}

func (p *Parser) parseBooleanLiteral() Node {
	return &BooleanLiteral{Token: p.CurToken, Value: p.CurTokenIs(token.TRUE)}
}

func (p *Parser) parseBreak() Node {
	if p.isPositionallyFunctional() {
		t := p.CurToken
		p.NextToken()                  // Skips the 'break' token
		exp := p.ParseExpression(FUNC) // If this is a multiple return, we don't want its elements to be treated as parameters of a function. TODO --- gve 'break' its own node type?
		return &PrefixExpression{t, "break", []Node{exp}}
	}
	return &Identifier{Token: p.CurToken, Value: "break"}
}

// This is to allow me to use the initializer to pour builtins into the parser's function table.
func (p *Parser) parseBuiltInExpression() Node {
	expression := &BuiltInExpression{}
	expression.Token = p.CurToken
	p.NextToken()
	if p.CurToken.Type == token.STRING {
		expression.Name = p.CurToken.Literal
	} else {
		panic("Expecting a string after 'builtin'.")
	}
	p.NextToken()
	return expression
}

func (p *Parser) parseComparisonExpression(left Node) Node {
	expression := &ComparisonExpression{
		Token:    p.CurToken,
		Operator: p.CurToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.NextToken()
	expression.Right = p.ParseExpression(precedence)
	return expression
}

func (p *Parser) parseContinue() Node {
	return &Identifier{Token: p.CurToken, Value: "continue"}
}

func (p *Parser) parseElse() Node {
	return &BooleanLiteral{Token: p.CurToken, Value: true}
}

// The fact that it is a valid float has been checked by the lexer.
func (p *Parser) parseFloatLiteral() Node {
	fVal, _ := strconv.ParseFloat(p.CurToken.Literal, 64)
	return &FloatLiteral{Token: p.CurToken, Value: fVal}
}

func (p *Parser) parseForAsInfix(left Node) *ForExpression {
	expression := p.parseForExpression()
	expression.BoundVariables = left
	return expression
}

func (p *Parser) parseForExpression() *ForExpression {
	expression := &ForExpression{
		Token: p.CurToken,
	}
	p.NextToken()
	// We handle the 'for :' as "while true" case.
	if p.CurToken.Type == token.COLON {
		p.NextToken()
		expression.Body = p.ParseExpression(COLON)
		return expression
	}

	pieces := p.ParseExpression(GIVEN)
	if pieces.GetToken().Type == token.COLON {
		expression.Body = pieces.(*LazyInfixExpression).Right
		header := pieces.(*LazyInfixExpression).Left
		if header.GetToken().Type == token.MAGIC_SEMICOLON { // If it has one, it should have two.
			leftBitOfHeader := header.(*InfixExpression).Args[0]
			rightBitOfHeader := header.(*InfixExpression).Args[2]
			if leftBitOfHeader.GetToken().Type == token.MAGIC_SEMICOLON {
				expression.Initializer = leftBitOfHeader.(*InfixExpression).Args[0]
				expression.ConditionOrRange = leftBitOfHeader.(*InfixExpression).Args[2]
				expression.Update = rightBitOfHeader
			} else {
				p.Throw("parse/for/semicolon", &expression.Token)
				return nil
			}
		} else {
			expression.ConditionOrRange = header
		}
	} else {
		p.Throw("parse/for/colon", &expression.Token)
		return nil
	}
	return expression
}

func (p *Parser) parseFromExpression() Node {
	fromToken := p.CurToken
	p.NextToken()
	expression := p.ParseExpression(FUNC)
	var givenBlock Node
	if expression.GetToken().Type == token.GIVEN {
		givenBlock = expression.(*InfixExpression).Args[2]
		expression = expression.(*InfixExpression).Args[0]
	}
	exp, ok := expression.(*ForExpression)
	if ok {
		exp.Given = givenBlock
		return exp
	}
	p.Throw("parse/from", &fromToken)
	return nil
}

func (p *Parser) parseGolangExpression() Node {
	expression := &GolangExpression{
		Token: p.CurToken,
	}
	p.NextToken()
	return expression
}

func (p *Parser) parseGroupedExpression() Node {
	p.NextToken()
	if p.CurToken.Type == token.RPAREN { // Then what we must have is an empty tuple.
		return &Nothing{Token: p.CurToken}
	}
	exp := p.ParseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		p.NextToken() // Forces emission of the error.
		return nil
	}
	return exp
}

func (p *Parser) parseIdentifier() Node {
	return &Identifier{Token: p.CurToken, Value: p.CurToken.Literal}
}

func (p *Parser) parseIfLogExpression(left Node) Node {
	expression := &LogExpression{
		Token: p.CurToken,
		Left:  left,
		Value: p.CurToken.Literal,
	}
	precedence := p.curPrecedence()
	p.NextToken()
	expression.Right = p.ParseExpression(precedence)
	return expression
}

func (p *Parser) parseIndexExpression(left Node) Node {
	exp := &IndexExpression{Token: p.CurToken, Left: left}
	p.NextToken()
	exp.Index = p.ParseExpression(LOWEST)
	if !p.expectPeek(token.RBRACK) {
		p.NextToken() // Forces emission of error
		return nil
	}
	return exp
}

func (p *Parser) parseInfixExpression(left Node) Node {
	if assignmentTokens.Contains(p.CurToken.Type) {
		return p.parseAssignmentExpression(left)
	}
	// TODO --- NOTE. This is basically the Last Of The Shotgun Parsing and there's no reason why the whole species shouldn't go extinct.
	if p.CurToken.Type == token.MAGIC_COLON {
		// Then we will magically convert a function declaration into an assignment of a lambda to a
		// constant.
		newTok := p.CurToken
		newTok.Type = token.GVN_ASSIGN
		newTok.Literal = "="
		p.NextToken()
		right := p.ParseExpression(FUNC)
		fn := &FuncExpression{Token: newTok}
		expression := &AssignmentExpression{Token: newTok}
		switch left := left.(type) {
		case *PipingExpression:
			if left.GetToken().Literal != "->" {
				p.Throw("parse/inner/a", left.GetToken())
			}
			fn.NameRets = p.RecursivelySlurpReturnTypes(left.Right)
			switch newLeft := left.Left.(type) {
			case *PrefixExpression:
				expression.Left = &Identifier{Token: *newLeft.GetToken(), Value: newLeft.GetToken().Literal}
				fn.NameSig, _ = p.getSigFromArgs(newLeft.Args, ANY_NULLABLE_TYPE_AST)
			default:
				p.Throw("parse/inner/b", newLeft.GetToken())
			}
		case *PrefixExpression:
			expression.Left = &Identifier{Token: *left.GetToken(), Value: left.GetToken().Literal}
			fn.NameSig, _ = p.getSigFromArgs(left.Args, ANY_NULLABLE_TYPE_AST)
		default:
			p.Throw("parse/inner/c", left.GetToken())
			return nil
		}
		if right.GetToken().Type == token.GIVEN {
			fn.Body = right.(*InfixExpression).Args[0]
			fn.Given = right.(*InfixExpression).Args[2]
		} else {
			fn.Body = right
		}
		expression.Right = fn
		return expression
	}
	expression := &InfixExpression{
		Token:    p.CurToken,
		Operator: p.CurToken.Literal,
	}
	precedence := p.curPrecedence()
	p.NextToken()
	right := p.ParseExpression(precedence)
	if expression.Operator == "," {
		expression.Args = []Node{left, &Bling{Value: expression.Operator, Token: expression.Token}, right}
		return expression
	}
	expression.Args = p.RecursivelyListify(left)
	expression.Args = append(expression.Args, &Bling{Value: expression.Operator, Token: expression.Token})
	rightArgs := p.RecursivelyListify(right)
	expression.Args = append(expression.Args, rightArgs...)
	return expression
}

func (p *Parser) parseIntegerLiteral() Node {
	iVal, _ := strconv.Atoi(p.CurToken.Literal)
	return &IntegerLiteral{Token: p.CurToken, Value: iVal}
}

func (p *Parser) parseLambdaExpression() Node {
	expression := &FuncExpression{
		Token: p.CurToken,
	}
	p.NextToken()
	RHS := p.ParseExpression(FUNC)
	// At this point the root of the RHS should be the colon dividing the function sig from its body.
	root := RHS
	var given Node 
	if root.GetToken().Type == token.GIVEN {
		root = RHS.(*InfixExpression).Args[0]
		given = RHS.(*InfixExpression).Args[2]
	}
	if root.GetToken().Type != token.COLON {
		p.Throw("parse/colon", &p.CurToken)
		return nil
	}
	LHS := root.(*LazyInfixExpression).Left
	var returns Node
	sig := LHS
	if LHS.GetToken().Type == token.PIPE {
		sig = LHS.(*PipingExpression).Left 
		returns = LHS.(*PipingExpression).Right
	}
	expression.NameSig, _ = p.ReparseSig(sig, ANY_NULLABLE_TYPE_AST)
	expression.NameRets = p.RecursivelySlurpReturnTypes(returns)
	bodyRoot := root.(*LazyInfixExpression).Right
	expression.Body = bodyRoot
	expression.Given = given
	return expression
}

// I.e `and`, `or`, `:`, and `;`.
func (p *Parser) parseLazyInfixExpression(left Node) Node {
	expression := &LazyInfixExpression{
		Token:    p.CurToken,
		Operator: p.CurToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.NextToken()
	expression.Right = p.ParseExpression(precedence)
	return expression
}

func (p *Parser) parseListExpression() Node {
	p.NextToken()
	if p.CurToken.Type == token.RBRACK { // Deals with the case where the list is []
		return &ListExpression{List: &Nothing{Token: p.CurToken}, Token: p.CurToken}
	}
	exp := p.ParseExpression(LOWEST)
	if !p.expectPeek(token.RBRACK) {
		p.NextToken() // Forces emission of error.
		return nil
	}
	expression := &ListExpression{List: exp, Token: p.CurToken}
	return expression
}

func (p *Parser) parseLogExpression(left Node) Node {
	expression := &LogExpression{
		Token: p.CurToken,
		Left:  left,
		Value: p.CurToken.Literal,
	}
	return expression
}

// For things like NOT, UNWRAP, VALID where we don't want to treat it as a function but to evaluate the RHS and then handle it.
func (p *Parser) parseNativePrefixExpression() Node {
	expression := &PrefixExpression{
		Token:    p.CurToken,
		Operator: p.CurToken.Literal,
	}
	prefix := p.CurToken
	p.NextToken()
	right := p.ParseExpression(precedences[prefix.Type])
	if right == nil {
		p.Throw("parse/follow", &prefix)
	}
	expression.Args = []Node{right}
	return expression
}

func (p *Parser) parsePrefixExpression() Node {
	expression := &PrefixExpression{
		Token:    p.CurToken,
		Operator: p.CurToken.Literal,
	}
	p.NextToken()
	var right Node
	if p.CurToken.Type == token.LPAREN || expression.Operator == "-" {
		right = p.ParseExpression(MINUS)
	} else {
		right = p.ParseExpression(FPREFIX)
	}
	expression.Args = p.RecursivelyListify(right)
	return expression
}

func (p *Parser) parsePrelogExpression() Node {

	expression := &LogExpression{
		Token: p.CurToken,
		Value: p.CurToken.Literal,
	}
	precedence := p.curPrecedence()
	p.NextToken()
	expression.Right = p.ParseExpression(precedence)
	return expression
}

func (p *Parser) parseRuneLiteral() Node {
	r, _ := utf8.DecodeRune([]byte(p.CurToken.Literal)) // We have already checked that the literal is a any rune at the lexing stage.
	return &RuneLiteral{Token: p.CurToken, Value: r}
}

// In a streaming expression we need to desugar e.g. 'x -> foo' to 'x -> foo that', etc.
func (p *Parser) parseStreamingExpression(left Node) Node {
	expression := &PipingExpression{
		Token:    p.CurToken,
		Operator: p.CurToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.NextToken()
	expression.Right = p.ParseExpression(precedence)
	expression.Right = p.recursivelyDesugarAst(expression.Right)
	return expression
}

// Function auxiliary to the previous one to get rid of syntactic sugar in streaming expressions.
// Adds "that" after piping, works through namespaces.
func (p *Parser) recursivelyDesugarAst(exp Node) Node {
	switch typedExp := exp.(type) {
	case *Identifier:
		if p.Functions.Contains(exp.GetToken().Literal) {
			exp = &PrefixExpression{Token: *typedExp.GetToken(),
				Operator: exp.GetToken().Literal,
				Args:     []Node{&Identifier{Value: "that"}}}
		}
		ok, rp := p.CanParse(*exp.GetToken(), SUFFIX)
		if rp == nil {
			return nil
		}
		if ok {
			exp = &SuffixExpression{Token: *typedExp.GetToken(),
				Operator: exp.GetToken().Literal,
				Args:     []Node{&Identifier{Value: "that"}}}
		}
	}
	return exp
}

func (p *Parser) parseSnippetExpression(tok token.Token) Node {
	codeTokens := p.TokenizedCode
	nesting := p.nesting.Copy()
	cT := p.CurToken
	pT := p.PeekToken
	nodes := []Node{}
	bits, ok := text.GetTextWithBarsAsList(tok.Literal)
	if !ok {
		p.Throw("parse/snippet/form", &tok)
		return nil
	}
	if len(bits) > 0 && len(bits[0]) > 0 && bits[0][0] == '|' {
		bits = append([]string{""}, bits...)
	}
	for i, bit := range bits {
		if i%2 == 0 {
			nodes = append(nodes, &StringLiteral{tok, bit})
		} else {
			p.nesting = dtypes.Stack[token.Token]{}
			node := p.ParseLine("embedded Pipefish in snippet", bit[1:len(bit)-1])
			nodes = append(nodes, node)
		}
	}
	p.nesting = *nesting
	p.TokenizedCode = codeTokens
	p.CurToken = cT
	p.PeekToken = pT
	return &SnippetLiteral{Token: tok, Value: tok.Literal, Values: nodes}
}

func (p *Parser) parseStringLiteral() Node {
	return &StringLiteral{Token: p.CurToken, Value: p.CurToken.Literal}
}

func (p *Parser) parseSuffixExpression(left Node) Node {
	expression := &SuffixExpression{
		Token:    p.CurToken,
		Operator: p.CurToken.Literal,
		Args:     p.RecursivelyListify(left),
	}
	return expression
}

func (p *Parser) parseTryExpression() Node {
	p.NextToken()
	if p.CurToken.Type == token.COLON {
		p.NextToken()
		exp := p.ParseExpression(COLON)
		return &TryExpression{Token: p.CurToken, Right: exp, VarName: ""}
	}
	if p.CurToken.Type == token.IDENT {
		varName := p.CurToken.Literal
		p.NextToken()
		if p.CurToken.Type != token.COLON {
			p.Throw("parse/try/colon", &p.CurToken)
		}
		p.NextToken()
		exp := p.ParseExpression(COLON)
		return &TryExpression{Token: p.CurToken, Right: exp, VarName: varName}
	} else {
		p.Throw("parse/try/ident", &p.CurToken)
		return nil
	}
}

func (p *Parser) parseUnfixExpression() Node {
	return &UnfixExpression{Token: p.CurToken, Operator: p.CurToken.Literal}
}

// This takes the arguments at the call site of a function and puts them
// into a list for us.
func (p *Parser) RecursivelyListify(start Node) []Node {
	switch start := start.(type) {
	case *InfixExpression:
		if start.Operator == "," {
			left := p.RecursivelyListify(start.Args[0])
			left = append(left, p.RecursivelyListify(start.Args[2])...)
			return left
		}
		if p.Midfixes.Contains(start.Operator) {
			return start.Args
		}
	case *Nothing:
		return []Node{}
	}
	return []Node{start}
}

func (p *Parser) getParserFromNamespace(namespace []string) *Parser {
	lP := p
	for _, name := range namespace {
		s, ok := lP.NamespaceBranch[name]
		if ok {
			lP = s.Parser
			continue
		}
		p.Throw("parse/namespace/exists", &p.CurToken, name)
		return nil
	}
	// We don't need the resolving parser to parse anything but we *do* need to call positionallyFunctional,
	// so it needs the following data to work.
	lP.CurToken = p.CurToken
	lP.PeekToken = p.PeekToken
	return lP
}

// Some functions for interacting with a `TokenSupplier`.

func (p *Parser) NextToken() {
	p.checkNesting()
	p.SafeNextToken()
}

// This is used to prime the parser without triggering 'checkNesting'.
func (p *Parser) SafeNextToken() {
	if settings.SHOW_RELEXER && !(settings.IGNORE_BOILERPLATE && settings.ThingsToIgnore.Contains(p.CurToken.Source)) {
		println(text.PURPLE+p.CurToken.Type, p.CurToken.Literal+text.RESET)
	}
	p.CurToken = p.PeekToken
	p.PeekToken = p.TokenizedCode.NextToken()
}

// Function auxiliary to `NextToken` which will throw an error if the rules for nesting brackets are violated.
func (p *Parser) checkNesting() {
	if p.CurToken.Type == token.LPAREN || p.CurToken.Type == token.LBRACE ||
		p.CurToken.Type == token.LBRACK {
		p.nesting.Push(p.CurToken)
	}
	if p.CurToken.Type == token.RPAREN || p.CurToken.Type == token.RBRACE ||
		p.CurToken.Type == token.RBRACK {
		popped, poppable := p.nesting.Pop()
		if !poppable {
			p.Throw("parse/match", &p.CurToken)
			return
		}
		if !checkConsistency(popped, p.CurToken) {
			p.Throw("parse/nesting", &p.CurToken, &popped)
		}
	}
	if p.CurToken.Type == token.EOF {
		for popped, poppable := p.nesting.Pop(); poppable; popped, poppable = p.nesting.Pop() {
			p.Throw("parse/eol", &p.CurToken, &popped)
		}
	}
}

// A function auxiliary to the previous one to check whether a puported pair of brackets matches up.
func checkConsistency(left, right token.Token) bool {
	if left.Type == token.LPAREN && left.Literal == "(" &&
		right.Type == token.RPAREN && right.Literal == ")" {
		return true
	}
	if left.Type == token.LPAREN && left.Literal == "|->" &&
		right.Type == token.RPAREN && right.Literal == "<-|" {
		return true
	}
	if left.Type == token.LBRACK && right.Type == token.RBRACK {
		return true
	}
	if left.Type == token.LBRACE && right.Type == token.RBRACE {
		return true
	}
	return false
}

func (p *Parser) CurTokenIs(t token.TokenType) bool {
	return p.CurToken.Type == t
}

func (p *Parser) CurTokenMatches(t token.TokenType, s string) bool {
	return p.CurToken.Type == t && p.CurToken.Literal == s
}

func (p *Parser) PeekTokenIs(t token.TokenType) bool {
	return p.PeekToken.Type == t
}

func (p *Parser) PeekTokenMatches(t token.TokenType, s string) bool {
	return p.PeekToken.Type == t && p.PeekToken.Literal == s
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.PeekTokenIs(t) {
		p.NextToken()
		return true
	}
	return false
}

func (p *Parser) ParseTokenizedChunk() Node {
	p.SafeNextToken()
	p.SafeNextToken()
	expn := p.ParseExpression(LOWEST)
	p.NextToken()
	if p.CurToken.Type != token.EOF {
		p.Throw("parse/expected", &p.CurToken)
	}
	return expn
}

// Functions for dealing with Pipefish errors.

func (p *Parser) Throw(errorID string, tok *token.Token, args ...any) {
	if settings.SHOW_ERRORS {
		println(text.ORANGE + "Throwing error " + errorID + text.RESET)
	}
	c := *tok
	p.Common.Errors = err.Throw(errorID, p.Common.Errors, &c, args...)
}

func (p *Parser) ErrorsExist() bool {
	return len(p.Common.Errors) > 0
}

func (p *Parser) ReturnErrors() string {
	return err.GetList(p.Common.Errors)
}

func (p *Parser) ResetAfterError() {
	p.Common.Errors = []*err.Error{}
	p.ResetParser()
}

func (p *Parser) ResetParser() {
	p.nesting = dtypes.Stack[token.Token]{}
}

func newError(ident string, tok *token.Token, args ...any) *err.Error {
	errorToReturn := err.CreateErr(ident, tok, args...)
	errorToReturn.Trace = []*token.Token{tok}
	return errorToReturn
}
