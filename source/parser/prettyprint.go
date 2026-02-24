package parser

import (
	"bytes"
	"reflect"
	"strconv"

	"github.com/tim-hardcastle/pipefish/source/token"
)

type printFlavor int

const (
	ppOUTER printFlavor = iota
	ppINLINE
)

type printContext struct {
	indent      string
	flavor      printFlavor
	mustBracket bool
}

var inlineCtxt = printContext{"", ppINLINE, false}
var prefixCtxt = printContext{"", ppINLINE, true}

func (ctxt printContext) in() printContext {
	ctxt.indent = ctxt.indent + "    "
	return ctxt
}

func (p *Parser) PrettyPrintInline(node Node) string {
	return p.prettyPrint(node, inlineCtxt)
}

func (p *Parser) PrettyPrint(node Node) string {
	return p.prettyPrint(node, printContext{"", ppOUTER, false})
}

func (p *Parser) prettyPrint(node Node, ctxt printContext) string {
	var out bytes.Buffer
	out.WriteString(ctxt.indent)
	switch node := node.(type) {
	case *AssignmentExpression:
		out.WriteString(p.prettyPrint(node.Left, inlineCtxt))
		out.WriteString(" = ")
		out.WriteString(p.prettyPrint(node.Right, inlineCtxt))
	case *Bling:
		out.WriteString(node.Value)
	case *BooleanLiteral:
		out.WriteString(node.Token.Literal)
	case *ComparisonExpression:
		out.WriteString(p.prettyPrint(node.Left, inlineCtxt))
		out.WriteString(" ")
		out.WriteString(node.Operator)
		out.WriteString(" ")
		out.WriteString(p.prettyPrint(node.Right, inlineCtxt))
	case *FloatLiteral:
		out.WriteString(node.Token.Literal)
	case *ForExpression:
		if node.BoundVariables != nil {
			out.WriteString("from ")
			out.WriteString(p.prettyPrint(node.BoundVariables, inlineCtxt))
			out.WriteString(" ")
		}
		out.WriteString("for ")
		if node.Initializer != nil {
			out.WriteString(p.prettyPrint(node.Initializer, inlineCtxt))
			out.WriteString("; ")
		}
		if node.ConditionOrRange != nil {
			out.WriteString(p.prettyPrint(node.ConditionOrRange, inlineCtxt))
			if node.Update != nil {
				out.WriteString("; ")
			}
		}
		if node.Update != nil {
			out.WriteString(p.prettyPrint(node.Update, inlineCtxt))
		}
		out.WriteString(" :")
		switch ctxt.flavor {
		case ppOUTER:
			out.WriteString("\n")
			out.WriteString(p.prettyPrint(node.Body, ctxt.in()))
		case ppINLINE:
			out.WriteString(" ")
			out.WriteString(p.prettyPrint(node.Body, inlineCtxt))
		}
	case *FuncExpression:
		out.WriteString("func")
		out.WriteString(node.NameSig.String() + " :")
		switch ctxt.flavor {
		case ppOUTER:
			out.WriteString("\n")
			out.WriteString(p.prettyPrint(node.Body, ctxt.in()))
		case ppINLINE:
			out.WriteString(" ")
			out.WriteString(p.prettyPrint(node.Body, inlineCtxt))
		}
		if node.Given != nil {
			switch ctxt.flavor {
			case ppOUTER:
				out.WriteString("given :\n")
				out.WriteString(p.prettyPrint(node.Body, ctxt.in()))
				out.WriteString("\n")
			case ppINLINE:
				out.WriteString("given : ")
				out.WriteString(p.prettyPrint(node.Body, inlineCtxt))
			}
		}
	case *Identifier:
		out.WriteString(node.Value)
	case *IndexExpression:
		if !isLeaf(node.Left) {
			out.WriteString("(")
		}
		out.WriteString(p.prettyPrint(node.Left, inlineCtxt))
		if !isLeaf(node.Left) {
			out.WriteString(")")
		}
		out.WriteString("[")
		out.WriteString(p.prettyPrint(node.Index, inlineCtxt))
		out.WriteString("]")
	case *InfixExpression:
		pos := 1
		if len(node.Args) != 3 {
			for ; pos <= len(node.Args); pos++ {
				if blingNode, ok := node.Args[pos].(*Bling); ok && blingNode.Value == node.Operator { // TODO --- record this in ast?
					break
				}
			}
		}
		_, isPrefix := node.Args[0].(*PrefixExpression)
		_, isList := node.Args[0].(*ListExpression)
		leftNeedsPrefix := isPrefix && pos == 1
		leftNeedsBrackets := !leftNeedsPrefix && !isList && (pos > 1 || p.hasLowerPrecedence(node.Args[0], node.Args[1]) && !isLeaf(node.Args[0]))
		rhsHasBling := false
		for i := pos + 1; i < len(node.Args); i++ {
			if _, ok := node.Args[i].(*Bling); ok {
				rhsHasBling = true
				break
			}
		}
		rightNeedsBrackets := !rhsHasBling && (len(node.Args)-pos) > 2 || p.hasHigherOrEqualPrecedence(node.Args[pos], node.Args[pos+1]) && !isLeaf(node.Args[pos+1])
		if leftNeedsBrackets {
			out.WriteString("(")
		}
		sep := ""
		for i := 0; i < pos; i++ {
			out.WriteString(sep)
			if leftNeedsPrefix {
				out.WriteString(p.prettyPrint(node.Args[i], prefixCtxt))
			} else {
				out.WriteString(p.prettyPrint(node.Args[i], inlineCtxt))
			}
			sep = ", "
		}
		if leftNeedsBrackets {
			out.WriteString(")")
		}
		if node.Operator != "," && node.Operator != "::" {
			out.WriteString(" ")
		}
		out.WriteString(node.Operator)
		if node.Operator != "::" {
			out.WriteString(" ")
		}
		if rightNeedsBrackets {
			out.WriteString("(")
		}
		for i := pos + 1; i < len(node.Args); i++ {
			if IsBling(node.Args[i-1]) || IsBling(node.Args[i]) {
				if i != pos+1 {
					out.WriteString(" ")
				}
			} else {
				out.WriteString(", ")
			}
			out.WriteString(p.prettyPrint(node.Args[i], inlineCtxt))
		}
		if rightNeedsBrackets {
			out.WriteString(")")
		}
	case *IntegerLiteral:
		out.WriteString(node.Token.Literal)
	case *LazyInfixExpression:
		if node.Operator == "and" || node.Operator == "or" || ctxt.flavor == ppINLINE {
			if p.hasLowerPrecedence(node.Left, node) && !isLeaf(node.Left) {
				out.WriteString("(")
			}
			out.WriteString(p.prettyPrint(node.Left, inlineCtxt))
			if p.hasLowerPrecedence(node.Left, node) && !isLeaf(node.Left) {
				out.WriteString(")")
			}
			out.WriteString(" ")
			out.WriteString(node.Operator)
			out.WriteString(" ")
			if p.hasHigherOrEqualPrecedence(node, node.Right) && !isLeaf(node.Right) {
				out.WriteString("(")
			}
			out.WriteString(p.prettyPrint(node.Right, inlineCtxt))
			if p.hasHigherOrEqualPrecedence(node, node.Right) && !isLeaf(node.Right) {
				out.WriteString(")")
			}
		} else {
			if node.Token.Type == token.SEMICOLON || node.Token.Type == token.NEWLINE {
				out.WriteString(p.prettyPrint(node.Left, ctxt))
				out.WriteString("\n")
				out.WriteString(p.prettyPrint(node.Right, ctxt))
			}
			if node.Token.Type == token.COLON {
				out.WriteString(p.prettyPrint(node.Left, inlineCtxt))
				out.WriteString(" :\n")
				out.WriteString(p.prettyPrint(node.Right, ctxt.in()))
			}
		}
	case *ListExpression:
		out.WriteString("[")
		out.WriteString(p.prettyPrint(node.List, inlineCtxt))
		out.WriteString("]")
	case *Nothing:
		out.WriteString("()")
	case *PipingExpression:
		_, isList := node.Left.(*ListExpression)
		leftNeedsBrackets := p.hasLowerPrecedence(node.Left, node) && !isList && !isLeaf(node.Left)
		if leftNeedsBrackets {
			out.WriteString("(")
		}
		out.WriteString(p.prettyPrint(node.Left, inlineCtxt))
		if leftNeedsBrackets {
			out.WriteString(")")
		}
		out.WriteString(" ")
		out.WriteString(node.Operator)
		out.WriteString(" ")
		if p.hasHigherOrEqualPrecedence(node, node.Right) && !isLeaf(node.Right) {
			out.WriteString("(")
		}
		out.WriteString(p.prettyPrint(node.Right, inlineCtxt))
		if p.hasHigherOrEqualPrecedence(node, node.Right) && !isLeaf(node.Right) {
			out.WriteString(")")
		}
	case *PrefixExpression:
		out.WriteString(node.Operator)
		if len(node.Args) == 0 {
			out.WriteString("()")
			break
		}
		if ctxt.mustBracket {
			out.WriteString("(")
		} else {
			out.WriteString(" ")
		}
		for i, arg := range node.Args {
			if i == 0 {
				if len(node.Args) > 1 {
					out.WriteString(p.prettyPrint(arg, prefixCtxt))
					if IsBling(arg) {
						out.WriteString(" ")
					}
				} else {
					out.WriteString(p.prettyPrint(arg, inlineCtxt))
				}
				continue
			}
			if IsBling(arg) {
				if ctxt.mustBracket && !IsBling(node.Args[i-1]) {
					out.WriteString(") ")
				}
				if !IsBling(node.Args[i-1]) {
					out.WriteString(" ")
				}
				out.WriteString(p.prettyPrint(arg, inlineCtxt))
				if i < len(node.Args)-1 {
					out.WriteString(" ")
				}
				if ctxt.mustBracket && i+1 < len(node.Args) && !IsBling(node.Args[i+1]) {
					out.WriteString("(")
				}
			} else {
				if !IsBling(node.Args[i-1]) {
					out.WriteString(", ")
				}
				if i+1 < len(node.Args) {
					out.WriteString(p.prettyPrint(arg, prefixCtxt))
				} else {
					out.WriteString(p.prettyPrint(arg, inlineCtxt))
				}
			}
		}
		if ctxt.mustBracket {
			out.WriteString(")")
		}
	case *RuneLiteral:
		out.WriteString(strconv.QuoteRune(node.Value))
	case *StringLiteral:
		out.WriteString(strconv.Quote(node.Value))
	case *SuffixExpression:
		if len(node.Args) > 1 || !isLeaf(node.Args[0]) {
			out.WriteString("(")
		}
		sep := ""
		for i := 0; i < len(node.Args); i++ {
			out.WriteString(sep)
			out.WriteString(p.prettyPrint(node.Args[i], inlineCtxt))
			sep = ", "
		}
		if len(node.Args) > 1 || !isLeaf(node.Args[0]) {
			out.WriteString(")")
		} else {
			out.WriteString(" ")
		}
		out.WriteString(node.Operator)
	case *TypeExpression:
		out.WriteString(node.Operator)
		if len(node.TypeArgs) == 0 {
			break
		}
		out.WriteString("{")
		sep := []byte("")
		for _, arg := range node.TypeArgs {
			out.Write(sep)
			out.WriteString(p.prettyPrint(arg, prefixCtxt))
			sep = []byte(", ")
		}
		out.WriteString("}")
	case *TypePrefixExpression:
		out.WriteString(node.Operator)
		if len(node.TypeArgs) > 0 {
			out.WriteString("{")
			sep := []byte("")
			for _, arg := range node.TypeArgs {
				out.Write(sep)
				out.WriteString(p.prettyPrint(arg, prefixCtxt))
				sep = []byte(", ")
			}
			out.WriteString("}")
		}
		out.WriteString("(")
		for i, arg := range node.Args {
			if i == 0 {
				if len(node.Args) > 1 {
					out.WriteString(p.prettyPrint(arg, prefixCtxt))
				} else {
					out.WriteString(p.prettyPrint(arg, inlineCtxt))
				}
				continue
			} else {
				out.WriteString(", ")
				if i+1 < len(node.Args) {
					out.WriteString(p.prettyPrint(arg, prefixCtxt))
				} else {
					out.WriteString(p.prettyPrint(arg, inlineCtxt))
				}
			}
		}
		out.WriteString(")")
	case *TryExpression:
		out.WriteString("try ")
		if node.VarName != "" {
			out.WriteString(node.VarName)
			out.WriteString(" ")
		}
		out.WriteString(":")
		switch ctxt.flavor {
		case ppOUTER:
			out.WriteString("\n")
			out.WriteString(p.prettyPrint(node.Right, ctxt.in()))
		case ppINLINE:
			out.WriteString(" ")
			out.WriteString(p.prettyPrint(node.Right, inlineCtxt))
		}
	case *UnfixExpression:
		out.WriteString(node.Operator)
	default:
		panic("Unhandled case in prettyprint: " + reflect.TypeOf(node).String())
	}
	return out.String()
}

func (p *Parser) hasLowerPrecedence(nodeA, nodeB Node) bool {
	return p.leftPrecedence(*nodeA.GetToken()) < p.rightPrecedence(*nodeB.GetToken())
}

func (p *Parser) hasHigherOrEqualPrecedence(nodeA, nodeB Node) bool {
	return p.leftPrecedence(*nodeA.GetToken()) >= p.rightPrecedence(*nodeB.GetToken())
}

func isLeaf(node Node) bool {
	return len(node.Children()) == 0
}
