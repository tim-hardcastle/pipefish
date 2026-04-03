package err_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

func TestAssignmentItes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`assign/type/a`, `OK`},
		{`assign/immutable`, `OK`},
	}
test_helper.RunTest(t, "test compiler errors", tests, test_helper.TestInitializationErrorsInCompiler)
}

func TestAssignmentErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`x = true`, `comp/typecheck/type`},
		{`y = "foo"`, `comp/typecheck/type`},
		{`y string = "foo"`, `comp/assign/type/b`},
		{`A string = "orange"`, `comp/assign/const`},
		{`x, y = 'q'`, `comp/typecheck/values/b`},
		{`x, y = 'q', 2, 3`, `comp/typecheck/values/b`},
	}
	test_helper.RunTest(t, "assignment_test.pf", tests, test_helper.TestCompilerErrors)
}

func TestBooleanCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`5 or true`, `comp/bool/or/left`},
		{`false or 5`, `comp/bool/or/right`},
		{`5 and false`, `comp/bool/and/left`},
		{`true and 5`, `comp/bool/and/right`},
		{`5 : 5`, `comp/bool/cond`},
		{`not 5`, `comp/bool/not`},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestBuiltinRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`7 / 0`, `vm/div/zero/a`},
		{`7.0 / 0.0`, `vm/div/zero/b`},
		{`7 div 0`, `vm/div/zero/c`},
		{`7.0 / 0`, `vm/div/zero/d`},
		{`7 / 0.0`, `vm/div/zero/e`},
		{`7 mod 0`, `vm/mod/zero`},
		{`map ([1]::2)`, `vm/map/key`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}
func TestCastRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`cast "foo", enum`, `vm/cast/concrete`},
		{`cast "foo", Person`, `vm/cast`},
		{`cast -1, Color`, `vm/cast/enum`},
		{`cast 99, Color`, `vm/cast/enum`},
		{`cast ["John", 22, true], Person`, `vm/cast/fields`},
		{`cast ["John", "22"], Person`, `vm/cast/types`},
		{`float "foo"`, `vm/string/float`},
		{`int "foo"`, `vm/string/int`},
	}
	test_helper.RunTest(t, "cast_test.pf", tests, test_helper.TestValues)
}
func TestCompilerItes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`assign/type/a`, `OK`},
		{`fcis`, `OK`},
		{`for/bound/present`, `OK`},
		{`global/global`, `OK`},
		{`global/ident`, `OK`},
		{`try/return`, `OK`},
		{`try/var`, `OK`},
	}
	test_helper.RunTest(t, "test compiler errors", tests, test_helper.TestInitializationErrorsInCompiler)
}
func TestTChunkingItes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`abstract/ident`, `OK`},
		{`alias`, `OK`},
		{`clone/expect/b`, `OK`},
		{`clone/given`, `OK`},
		{`clone/type.c`, `OK`},
		{`enum/expect`, `OK`},
		{`enum/ident`, `OK`},
		{`impex/end`, `OK`},
		{`impex/expect`, `OK`},
		{`impex/pair`, `OK`},
		{`impex/string`, `OK`},
		{`interface/colon`, `OK`},
		{`struct/expect`, `OK`},
		{`struct/lparen`, `OK`},
		{`type/assign`, `OK`},
		{`type/expect.a`, `OK`},
		{`type/expect.b`, `OK`},
		{`type/ident`, `OK`},
		{`wrapper`, `OK`},
	}
	test_helper.RunTest(t, "test initialization errors", tests, test_helper.TestInitializationErrors)
}
func TestCloneRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`getClones 42`, `vm/clones/type`},
	}
	test_helper.RunTest(t, "clone_test.pf", tests, test_helper.TestValues)
}
func TestEnumRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Color 3`, `vm/enum`},
	}
	test_helper.RunTest(t, "enums_test.pf", tests, test_helper.TestValues)
}
func TestEqualityCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`(error "foo") == 42`, `comp/error/eq/a`},
		{`42 == (error "foo")`, `comp/error/eq/b`},
		{`zort someInt == troz someInt`, `comp/error/eq/c`},
		{`42 == "foo"`, `comp/eq/types`},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestEqualityRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`comp(foo(1), foo(2))`, `vm/equals/type`},
	}
	test_helper.RunTest(t, "equality_test", tests, test_helper.TestValues)
}
func TestForLoopCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`break 2 == true`, `comp/break/a`},
		{`from a == 0 for _::i = range 0::5 : a + i`, `comp/for/assign.a`},
		{`from a, a = 0, 0 for _::i = range z : a + i, a`, `comp/for/bound/exists`},
		{`from a = 0 for i == 0; i < 5; i + 1 : a + i`, `comp/for/assign.b`},
		{`from a = 0 for i, i = 0, 0; i < 5; i + 1 : a + i`, `comp/for/index/exists`},
		{`from a = 0 for true::i = range 0::5 : a + i`, `comp/for/range.a`},
		{`from a = 0 for i::true = range 0::5 : a + i`, `comp/for/range.b`},
		{`from a = 0 for i::j = range true : a + i`, `comp/for/range/types`},
		{`from a = 0 for a::j = range 5 : a + i`, `comp/for/exists/key`},
		{`from a = 0 for i::a = range 5 : a + i`, `comp/for/exists/value`},
		{`from a = 0 for i+j = range 5 : a + i`, `comp/for/range.c`},
		{`from a = 0 for i = 0; 1/0; i + 1 : a + i`, `comp/for/condition`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestCompilerErrors)
}
func TestForLoopRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`bar 5`, `vm/typecheck/bound/init`},
		{`foo 4`, `vm/typecheck/bound/update`},
		{`zort 3`, `vm/typecheck/index/init`},
		{`qux 3`, `vm/typecheck/index/update`},
		{`rozt 3`, `vm/types.a`},
		{`zrot 3`, `vm/types.a`},
		{`merp 3`, `vm/for/condition`},
		{`count any`, `vm/for/type/a`},
		{`count int`, `vm/for/type/b`},
		{`count true`, `vm/for/type/c`},
	}
	test_helper.RunTest(t, "for_loop_rtes_test.pf", tests, test_helper.TestValues)
}

func TestGivenCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`func(x) : x given: 42`, `comp/given/assign`},
		{`func(x) : x given: y = 1 div 0`, `comp/given/error`},
		{`func(x) : x given: x = 42`, `comp/given/exists`},
		//{"func(x) : x given:\n\ty = 42\n\ty = 42", `comp/given/redeclared`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestCompilerErrors)
}

func TestHubErrorMethods(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{"2 +", "[0] [31mError[39m: can't parse end of line as a prefix at line [33m1:3[39m of REPL input."},
		{`hub why 0`, "\x1b[31mError\x1b[39m: can't parse end of line as a prefix. \n\nYou've put end of line in such a position that it looks like you want it to function as a \x1b[0m\nprefix, but it isn't one. \x1b[0m\n\n                                                      Error has reference \x1b[0m\x1b[48;2;0;0;64m\x1b[97m\"parse/prefix\"\x1b[0m."},
		{`hub where 0`, "2 +\x1b[31m\n\x1b[0m   \x1b[31m▔\x1b[0m"},
		{`hub errors`, "[0] \x1b[31mError\x1b[39m: can't parse end of line as a prefix at line \x1b[33m1:3\x1b[39m of REPL input."},

		}
		test_helper.RunHubTest(t, "default", test)
	}
func TestIndexingCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`[1, 2, 3][4.0]`, `comp/index/list`},
		{`"foo"[4.0]`, `comp/index/string`},
		{`(1, 2, 3)[4.0]`, `comp/index/tuple`},
		{`(1::2)[4.0]`, `comp/index/pair`},
		{`SN[4.0]`, `comp/index/snippet`},
		{`JOHN[lives]`, "comp/index/struct/a"},
		{`JOHN[42]`, "comp/index/struct/b"},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestIndexingRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`[RED, GREEN, BLUE][true::2]`, `vm/slice/list/a`},
		{`[RED, GREEN, BLUE][2::true]`, `vm/slice/list/b`},
		{`[RED, GREEN, BLUE][-1::2]`, `vm/slice/list/c`},
		{`[RED, GREEN, BLUE][3::2]`, `vm/slice/list/d`},
		{`[RED, GREEN, BLUE][0::99]`, `vm/slice/list/e`},
		{`"aardvark"[true::2]`, `vm/slice/string/a`},
		{`"aardvark"[2::true]`, `vm/slice/string/b`},
		{`"aardvark"[-1::2]`, `vm/slice/string/c`},
		{`"aardvark"[3::2]`, `vm/slice/string/d`},
		{`"aardvark"[0::99]`, `vm/slice/string/e`},
		{`(1, 2, 3)[true::2]`, `vm/slice/tuple/a`},
		{`(1, 2, 3)[2::true]`, `vm/slice/tuple/b`},
		{`(1, 2, 3)[-1::2]`, `vm/slice/tuple/c`},
		{`(1, 2, 3)[3::2]`, `vm/slice/tuple/d`},
		{`(1, 2, 3)[0::99]`, `vm/slice/tuple/e`},
		{`ixE true, false`, `vm/user`},
		{`ixE false, true`, `vm/user`},
		{`myTuple[-1]`, `vm/index/m`},
		{`mySnippet[-1]`, `vm/index/s`},
		{`myList[-1]`, `vm/index/list`},
		{`myWord[-1]`, `vm/index/string`},
		{`myPair[-1]`, `vm/index/pair`},
		{`myTuple[99]`, `vm/index/m`},
		{`mySnippet[99]`, `vm/index/s`},
		{`myList[99]`, `vm/index/list`},
		{`myWord[99]`, `vm/index/string`},
		{`myPair[99]`, `vm/index/pair`},
		{`myMap[99]`, `vm/index/h`},
		{`myList["p"::0]`, `vm/slice/list/a`},
		{`myList[0::"q"]`, `vm/slice/list/b`},
		{`myTuple["p"::0]`, `vm/index/a`},
		{`myTuple[0::"q"]`, `vm/index/b`},
		{`myWord["p"::0]`, `vm/slice/string/a`},
		{`myWord[0::"q"]`, `vm/slice/string/b`},
		{`goo myTuple, -1`, `vm/index/m`},
		{`foo myList, -1`, `vm/index/j`},
		{`foo myWord, -1`, `vm/index/l`},
		{`foo myPair, -1`, `vm/index/k`},
		{`goo myTuple, 99`, `vm/index/m`},
		{`foo mySnippet, 99`, `vm/index/s`},
		{`foo myList, 99`, `vm/index/j`},
		{`foo myWord, 99`, `vm/index/l`},
		{`foo myPair, 99`, `vm/index/k`},
		{`foo myMap, 99`, `vm/index/h`},
		{`foo myList, "p"::0`, `vm/index/a`},
		{`foo myList, 0::"q"`, `vm/index/b`},
		{`goo myTuple, "p"::0`, `vm/index/a`},
		{`goo myTuple, 0::"q"`, `vm/index/b`},
		{`foo myWord, "p"::0`, `vm/index/a`},
		{`foo myWord, 0::"q"`, `vm/index/b`},
		{`foo myWord, -1::2`, `vm/index/c`},
		{`foo myWord, 3::2`, `vm/index/d`},
		{`foo myList, 1::99`, `vm/index/e`},
		{`foo myWord, 1::99`, `vm/index/f`},
		{`goo myTuple, 1::99`, `vm/index/r`},
		{`foo myBool, 1::99`, `vm/index/g`},
		{`foo myColor, charm`, `vm/index/t`},
		{`foo myColor, true`, `vm/index/label`},
		{`foo [1, 2, 3], "aardvark"`, `vm/index/i`},
		{`foo [1, 2, 3], -1`, `vm/index/j`},
		{`foo true, -1`, `vm/index/q`},
		{`ixs myColor, charm`, `vm/index/u`},
	}
	test_helper.RunTest(t, "index_test.pf", tests, test_helper.TestValues)
}
func TestInitializerItes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`depend/cmd`, `OK`},
		{`depend/var`, `OK`},
		{`make/ident`, `OK`},
		{`make/instance`, `OK`},
		{`name/exists.a`, `OK`},
		{`name/exists.b`, `OK`},
		{`overload.a`, `OK`},
		{`overload/ref`, `OK`},
		{`service/depends`, `OK`},
		{`service/type`, `OK`},
		{`typecheck/bool`, `OK`},
	}
	test_helper.RunTest(t, "test initialization errors", tests, test_helper.TestInitializationErrors)
}
func TestLabelRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`label "blerp"`, `vm/label/exists`},
	}
	test_helper.RunTest(t, "labels_test.pf", tests, test_helper.TestValues)
}

func TestLambdaCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`func(x) : x * y`, `comp/body/known`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestCompilerErrors)
}

func TestLoggingCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`42 \\ | forty-two`, `comp/log/close`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestCompilerErrors)
}
func TestMiscellaneousCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`[error "foo"]`, `comp/list/err`},
		{`break 42`, `comp/break/a`},
		{`break`, `comp/break/b`},
		{`continue`, `comp/continue`},
		{`w(42)`, `comp/apply/func`},
		{`("a", "b") ...`, `comp/splat/args`},
		{`"foo" ...`, `comp/splat/type`},
		{`42 >> that`, `comp/pipe/mf/list`},
		{`[1, 2, 3] ?> 2 * that`, `comp/pipe/filter/bool`},
		{`1 given : 2`, `comp/expect/given`},
		{`zwub 5`, `comp/known/prefix`},
		{`len(1/0)`, `comp/error/arg`},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestParameterizedTypeRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`fooify 1`, `vm/param/exist`},
	}
	test_helper.RunTest(t, "parameterized_type_test.pf", tests, test_helper.TestValues)
}
func TestParserErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`(2 + 2]`, `parse/close`},
		{`foo[2;`, `parse/prefix`},
		{`[2, 3, 4;`, `parse/prefix`},
		{`2 +`, `parse/prefix`},
		{`1 + )`, `parse/prefix`},
		{`1 + ]`, `parse/prefix`},
		{`len 1,`, `parse/prefix`},
		{`len(`, `parse/prefix`},
		{`len(1`, `parse/line`},
		{`troz.foo`, `parse/namespace/exists`},
		{`2 "aardvark"`, `parse/before.a`},
		{`func(x) wut`, `parse/colon`},
		{`from 1`, `parse/from`},
		{`(1))`, `parse/expected`},
		{`Z{5`, `parse/rbrace`},
		{`42 foo.bar`, `parse/namespace/exists`},
		{`42 foo.bar 99`, `parse/namespace/exists`},
		{`for i::j = range k`, `parse/for/colon`},
		{`for a = 1; a + 1 : foo`, `parse/for/semicolon`},
		{`func(x) >> int : x`, `parse/sig/c`},
		{`func(x int) : foo(x) given : foo(x int) >> int : x`, `parse/inner/a`},
		{`func(x int) : foo(x) given : foo(x int) == int : x`, `parse/inner/c`},
		{`not`, `parse/prefix`},
		{`-- foo |bar qux`, `parse/snippet/form`},
		{`try e @`, `parse/try/colon`},
		{`try 86 `, `parse/try/ident`},
	}
	test_helper.RunTest(t, "parser_error_test.pf", tests, test_helper.TestParserErrors)
}
func TestParsingItes(t *testing.T) {
	tests := []test_helper.TestItem{
		// {`clone/exists`, `OK`}, // TODO --- It doesn't throw this! It really should.
		{`clone/type.c`, `OK`},
		{`enum/element`, `OK`},
		{`head`, `OK`},
		{`import/file`, `OK`},
		{`label/exists`, `OK`},
		{`request/float`, `OK`},
		{`request/int`, `OK`},
		{`request/list`, `OK`},
		{`request/map`, `OK`},
		{`request/pair`, `OK`},
		{`request/rune`, `OK`},
		{`request/set`, `OK`},
		{`request/snippet`, `OK`},
		{`request/string`, `OK`},
	}
	test_helper.RunTest(t, "test initialization errors", tests, test_helper.TestInitializationErrors)
}
func TestTypeAccessCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Pair 1, 2`, `comp/private`},
		{`Suit`, `comp/private/type`},
		{`HEARTS`, `comp/ident/private`},
		{`one`, `comp/ident/private`},
	}
	test_helper.RunTest(t, "user_types_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestTypeExpressionCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`clones{string, int}`, `comp/clones/arguments`},
		{`clones{NULL}`, `comp/clones`},
		{`struct "foo"`, `comp/type/concrete`},
		{`("foo") list{int}`, `comp/suffix/b`},
		{`5 ; (post "foo")`, `comp/sanity`},
		{`qux.foo 42`, `comp/namespace/private`},
		{`(error "foo"), 42`, `comp/tuple/err/a`},
		{`42, (error "foo")`, `comp/tuple/err/b`},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestUnwrapRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`unwrap 42`, `vm/unwrap`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}
func TestValidationRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Thing 1`, `vm/validation/bool`},
		{`Thing 2`, `vm/validation/fail`},
	}
	test_helper.RunTest(t, "validation_test.pf", tests, test_helper.TestValues)
}
func TestVariableAccessCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`B`, `comp/ident/private`},
		{`A = 43`, `comp/assign/const`},
		{`z`, `comp/ident/private`},
		{`secretB`, `comp/private`},
		{`secretZ`, `comp/private`},
	}
	test_helper.RunTest(t, "variables_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestVariableCtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`i * i = 4`, `parse/sig/c`},
		{`w = 42`, `comp/assign/private`},
		{`X = 42`, `comp/assign/const`},
		{`noVar = 0`, `comp/assign/repl`},
		{`blerp`, `comp/ident/known`},
		{`w`, `comp/ident/private`},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestWithRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Addable with "foo"::99`, `vm/with/type/a`},
		{`int with "foo"::99`, `vm/with/type/b`},
		{`Person with "foo"::99`, `vm/with/type/d`},
		{`Person with friends::99`, `vm/with/type/e`},
		{`myList with true::"foo"`, `vm/with/a`},
		{`myList with -1::"foo"`, `vm/with/b`},
		{`myList with 6::"foo"`, `vm/with/b`},
		{`myMap with F::"foo"`, `vm/with/c`},
		{`Cat with (name::"John")`, `vm/with/type/g`},
		{`Cat with (name::"John", age::true)`, `vm/with/type/h`},
		{`john with []::23`, `vm/with/struct/b`},
		{`john with name::23`, `vm/with/f`},
		{`myOtherList with []::"q"`, `vm/with/list/b`},
		{`myMap with []::99`, `vm/with/map/b`},
	}
	test_helper.RunTest(t, "with_test.pf", tests, test_helper.TestValues)
}
