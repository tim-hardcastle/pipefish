package lexer

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/token"
)

func TestNextTokenForRelexer(t *testing.T) {
	input :=
		`foo(x):
	x : 1
	else : 2
`
	items := []testItem{
		{token.IDENT, "foo", 1},
		{token.LPAREN, "(", 1},
		{token.IDENT, "x", 1},
		{token.RPAREN, ")", 1},
		{token.COLON, ":", 1},
		{token.LPAREN, "|->", 2},
		{token.IDENT, "x", 2},
		{token.COLON, ":", 2},
		{token.INT, "1", 2},
		{token.NEWLINE, ";", 2},
		{token.ELSE, "else", 3},
		{token.COLON, ":", 3},
		{token.INT, "2", 3},
		{token.RPAREN, "<-|", 4},
		{token.EOF, "EOF", 4},
	}
	testRelexingString(t, input, items)
}

func TestRelexing(t *testing.T) {
	input :=
		`foo = func(x): 1 given : y = 2 ; qux(z) : 3`
	items := []testItem{
		{token.IDENT, "foo", 1},
		{token.ASSIGN, "=", 1},
		{token.IDENT, "func", 1},
		{token.LPAREN, "(", 1},
		{token.IDENT, "x", 1},
		{token.RPAREN, ")", 1},
		{token.COLON, ":", 1},
		{token.INT, "1", 1},
		{token.GIVEN, "given", 1},
		{token.IDENT, "y", 1},
		{token.GVN_ASSIGN, "=", 1},
		{token.INT, "2", 1},
		{token.SEMICOLON, ";", 1},
		{token.IDENT, "qux", 1},
		{token.LPAREN, "(", 1},
		{token.IDENT, "z", 1},
		{token.RPAREN, ")", 1},
		{token.COLON, ":", 1},
		{token.INT, "3", 1},
		{token.EOF, "EOF", 1},
	}
	testRelexingString(t, input, items)
}

func TestRelexingLogs(t *testing.T) {
	input :=
`foo(x): \\ zort
	true : \\ troz
		1
	else : 
		2`
	items := []testItem{
		{token.IDENT, "foo", 1},
		{token.LPAREN, "(", 1},
		{token.IDENT, "x", 1},
		{token.RPAREN, ")", 1},
		{token.COLON, ":", 1},
		{token.PRELOG, "zort", 1},
		{token.LPAREN, "|->", 2},
		{token.TRUE, "true", 2},
		{token.IFLOG, "troz", 2},
		{token.LPAREN, "|->", 3},
		{token.INT, "1", 3},
		{token.RPAREN, "<-|", 4},
		{token.NEWLINE, ";", 4},
		{token.ELSE, "else", 4},
		{token.COLON, ":", 4},
		{token.LPAREN, "|->", 5},
		{token.INT, "2", 5},
		{token.RPAREN, "<-|", 5},
		{token.RPAREN, "<-|", 5},
		{token.EOF, "EOF", 5},
	}
	testRelexingString(t, input, items)
}

func TestRlGolang(t *testing.T) {
	input :=

		`golang "qux"

golang {
    foo
}`

	items := []testItem{
		{token.GOLANG, "qux", 1},
		{token.NEWLINE, ";", 1},
		{token.GOLANG, "\n    foo\n", 3},
		{token.EOF, "EOF", 5},
	}
	testRelexingString(t, input, items)
}

func testRelexingString(t *testing.T, input string, items []testItem) {
	rl := NewRelexer("dummy source", input)
	runTest(t, rl, items)
}
