package parser_test

import (
	"errors"
	"testing"

	"github.com/tim-hardcastle/pipefish/source/compiler"
	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

func TestPrettyPrint(t *testing.T) {
	tests := []test_helper.TestItem{
		//{`func(x int) : x`, "func(x int) :\n    x"},
		{`func(x) : x`, "func(x any?) :\n    x"},
		{`2 + 2 == 4`, `2 + 2 == 4`},
		{`2 + 2 * 3`, `2 + 2 * 3`},
		{`true and (true or false)`, `true and (true or false)`},
		{`(true and true) or false`, `true and true or false`},
		{`"foo"[3]`, `"foo"[3]`},
		{`("foo" + "bar")[3]`, `("foo" + "bar")[3]`},
		{`[1, 2, 3]`, `[1, 2, 3]`},
		{`2 + 2`, `2 + 2`},
		{`foo 99`, `foo 99`},
		{`foo 99, 99`, `foo 99, 99`},
		{`foo(99) + 1`, `foo(99) + 1`},
		{`(foo 99, 99) + 1`, `foo(99, 99) + 1`},
		{`blerp`, `blerp`},
		{`moo boo 8`, `moo boo 8`},
		{`moo boo coo 8`, `moo boo coo 8`},
		{`moo zoo`, `moo zoo`},
		{`9 spoit`, `9 spoit`},
		{`xuq 9 mip`, `xuq 9 mip`},
		{`troz 8 nerf 9`, `troz 8 nerf 9`},
		{`goo 8 hoo 9 spoo 0`, `goo 8 hoo 9 spoo 0`},
		{`gee 8 hee 9 spee`, `gee 8 hee 9 spee`},
		{`gah 8 hah 9 spah blah`, `gah 8 hah 9 spah blah`},
		{`8 bing 9 bong`, `8 bing 9 bong`},
		{`8 ding 9 dong 0 dang`, `8 ding 9 dong 0 dang`},
		{`spong()`, `spong()`},
		{`[1, 2, 3] -> len`, `[1, 2, 3] -> len that`},
		{`len("foo") -> 2 + that`, `len "foo" -> 2 + that`},
		{`()`, `()`},
		{`bool`, `bool`},
		{`list{int}`, `list{int}`},
		{`int "5"`, `int("5")`},
		{`list{int}"5"`, `list{int}("5")`},
		{`list{int}("5", "3")`, `list{int}("5", "3")`},
		{`list{int, string}"5"`, `list{int, string}("5")`},
		{`'q'`, `'q'`},
		{`4.0`, `4.0`},
		{`true : 1 ; else : 2`, "true :\n    1\nelse :\n    2"},
		{`x = 99`, `x = 99`},
		{`from a = 0 for _::v = range L : a + v`, "from a = 0 for _::v = range L :\n    a + v"},
		{`from a = 0 for i = 0; i < n; i + 1 : a + i`, "from a = 0 for i = 0; i < n; i + 1 :\n    a + i"},
	}
	test_helper.RunTest(t, "prettyprint_test.pf", tests, testPrettyPrinter)
}

func TestBuiltins(t *testing.T) {
	tests := []test_helper.TestItem{
		{`2 + 2`, `(2 + 2)`},
		{`2 + 3 * 4`, `(2 + (3 * 4))`},
		{`2 * 3 + 4`, `((2 * 3) + 4)`},
		{`-5`, `(- 5)`},
		{`-5 + 3`, `((- 5) + 3)`},
		{`a + b + c`, `((a + b) + c)`},
		{`a + b - c`, `((a + b) - c)`},
		{`a * b * c`, `((a * b) * c)`},
		{`a * b / c`, `((a * b) / c)`},
		{`a + b / c`, `(a + (b / c))`},
		{`a + b[c]`, `(a + (b[c]))`},
		{`-a * b`, `((- a) * b)`},
		{`true or true and true`, `(true or (true and true))`},
		{`true and true or true`, `((true and true) or true)`},
		{`not x and not y`, `((not x) and (not y))`},
		{`1 + 2, 3 + 4`, `((1 + 2) , (3 + 4))`},
		{`1 < 2 == 3 < 4`, `((1 < 2) == (3 < 4))`},
		{`1 == 2 and 3 == 4`, `((1 == 2) and (3 == 4))`},
		{`1 == 2 or 3 == 4`, `((1 == 2) or (3 == 4))`},
		{`1 < 2 != 3 < 4`, `((1 < 2) != (3 < 4))`},
		{`1 != 2 and 3 <= 4`, `((1 != 2) and (3 <= 4))`},
		{`1 >= 2 or 3 > 4`, `((1 >= 2) or (3 > 4))`},
		{`2 + 2 == 4 and true`, `(((2 + 2) == 4) and true)`},
		{`1 + 2 < 3 + 4`, `((1 + 2) < (3 + 4))`},
		{`1 * 2 > 3 mod 4`, `((1 * 2) > (3 mod 4))`},
		{`x = func(y) : y * y`, `(x = func (y any?) : (y * y))`},
		{`from a for i = 1; i < n; i + 1 : a + i`, `from a for (i = 1); (i < n); (i + 1) : (a + i)`},
		{`len x`, `(len x)`},
		{`len x, y`, `(len x, y)`},
		{`len(x), y`, `((len x) , y)`},
		{`x in Y, Z`, `(x in Y, Z)`},
		{`v + w :: x + y`, `((v + w) :: (x + y))`},
		{`x in int`, `(x in int)`},
		{`x -> y`, `(x -> y)`},
		{`[1, 2, 3]`, `[((1 , 2) , 3) ]`},
		{`'q'`, `'q'`},
		{`0.42`, `0.42`},
		{`valid(x)`, `(valid x)`},
		{`unwrap(x)`, `(unwrap x)`},
		{`break`, `break`},
		{`break 42`, `(break 42)`},
		{`continue`, `continue`},
		{`true : 42 ; else : "moo!"`, `((true : 42) ; (else : "moo!"))`},
	}
	test_helper.RunTest(t, "", tests, testParserOutput)
}
func TestFunctionSyntax(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo x`, `(foo x)`},
		{`x zort`, `(x zort)`},
		{`x troz y`, `(x troz y)`},
		{`moo x goo`, `(moo x goo)`},
		{`flerp x blerp y`, `(flerp x blerp y)`},
		{`qux`, `(qux)`},
	}
	test_helper.RunTest(t, "function_syntax_test.pf", tests, testParserOutput)
}
func TestFancyFunctionSyntax(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 99`, `(foo 99)`},
		{`blerp`, `(blerp)`},
		{`moo boo 8`, `(moo boo 8)`},
		{`moo boo coo 8`, `(moo boo coo 8)`},
		{`moo zoo`, `(moo zoo)`},
		{`9 spoit`, `(9 spoit)`},
		{`xuq 9 mip`, `(xuq 9 mip)`},
		{`troz 8 nerf 9`, `(troz 8 nerf 9)`},
		{`goo 8 hoo 9 spoo 0`, `(goo 8 hoo 9 spoo 0)`},
		{`gee 8 hee 9 spee`, `(gee 8 hee 9 spee)`},
		{`gah 8 hah 9 spah blah`, `(gah 8 hah 9 spah blah)`},
		{`8 bing 9 bong`, `(8 bing 9 bong)`},
		{`8 ding 9 dong 0 dang`, `(8 ding 9 dong 0 dang)`},
		{`spong()`, `(spong ())`},
	}
	test_helper.RunTest(t, "fancy_function_test.pf", tests, testParserOutput)
}

func TestSnippets(t *testing.T) {
	tests := []test_helper.TestItem{
		{`-- foo |bar| qux`, `(-- foo |bar| qux)`},
		{`true -- foo |bar| qux`, `(true , (-- foo |bar| qux))`},
	}
	test_helper.RunTest(t, "function_syntax_test.pf", tests, testParserOutput)
}
func TestTypeParser(t *testing.T) {
	tests := []test_helper.TestItem{
		{`string/int`, `string/int`},
		{`string&int`, `string&int`},
		{`string`, `string`},
		{`int?`, `int?`},
		{`int!`, `int!`},
		{`string{42}`, `string{42}`},
		{`string{42, 43}`, `string{42, 43}`},
		{`string{true}`, `string{true}`},
		{`string{4.2}`, `string{4.2}`},
		{`string{"foo"}`, `string{"foo"}`},
		{`string{'q'}`, `string{'q'}`},
		{`list{T type}`, `list{T type}`},
		{`pair{K, V type}`, `pair{K type, V type}`},
		{`list{string}`, `list{string}`},
		{`list{list{string}}`, `list{list{string}}`},
		{`clones{int}/string`, `clones{int}/string`},
		{`clones{int}/clones{string}`, `clones{int}/clones{string}`},
	}
	test_helper.RunTest(t, "", tests, testTypeParserOutput)
}

func TestParserErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`2 +`, `parse/prefix`},
		{`1 + )`, `parse/prefix`},
		{`1 + ]`, `parse/prefix`},
		{`len 1,`, `parse/prefix`},
		{`len(`, `parse/prefix`},
		{`len(1`, `parse/line`},
		{`troz.foo`, `parse/namespace/exists`},
		{`2 "aardvark"`, `parse/before/a`},
		{`func(x) wut`, `parse/colon`},
		{`from 1`, `parse/from`},
		{`(1))`, `parse/expected`},
	}
	test_helper.RunTest(t, "", tests, testParserErrors)
}

// The helper functions for testing the parser.

func testParserOutput(cp *compiler.Compiler, s string) (string, error) {
	astOfLine := cp.P.ParseLine("test", s)
	if cp.P.ErrorsExist() {
		return "", errors.New("compilation error")
	}
	return astOfLine.String(), nil
}

func testPrettyPrinter(cp *compiler.Compiler, s string) (string, error) {
	astOfLine := cp.P.ParseLine("test", s)
	if cp.P.ErrorsExist() {
		return "", errors.New("compilation error")
	}
	return cp.P.PrettyPrint(astOfLine), nil
}

func testTypeParserOutput(cp *compiler.Compiler, s string) (string, error) {
	astOfLine := cp.P.ParseTypeFromString(s)
	if cp.P.ErrorsExist() {
		return "", errors.New("compilation error")
	}
	if astOfLine == nil {
		return "nil", nil
	}
	return astOfLine.String(), nil
}

func testParserErrors(cp *compiler.Compiler, s string) (string, error) {
	cp.P.ParseLine("test", s)
	if cp.P.ErrorsExist() {
		return cp.P.Common.Errors[0].ErrorId, nil
	} else {
		return "", errors.New("unexpected successful parsing")
	}
}
