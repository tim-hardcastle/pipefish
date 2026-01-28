package compiler_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

func TestAlias(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Strings == list{string}`, `true`},
		{`Strings["foo", "bar"] == list{string}["foo", "bar"]`, `true`},
		{`OtherList["foo", "bar"] == ["foo", "bar"]`, `true`},
		{`OtherFoo("foo", "bar") == Foo("foo", "bar")`, `true`},
	}
	test_helper.RunTest(t, "alias_test.pf", tests, test_helper.TestValues)
}

func TestAssignment(t *testing.T) {
	tests := []test_helper.TestItem{
		{`x`, `'q'`},
		{`y`, `2`},
		{`x rune, y int = 'z', 42`, `OK`},
		{`y = 42`, `OK`},
	}
	test_helper.RunTest(t, "assignment_test.pf", tests, test_helper.TestValues)
}

// func TestAssignmentErrors(t *testing.T) {
// 	tests := []test_helper.TestItem{
// 		{`x = true`, `OK`},
// 		{`y = "foo"`, `OK`},
// 	}
// 	test_helper.RunTest(t, "assignment_test.pf", tests, test_helper.TestValues)
// }

func TestAssignmentErrorsInSource(t *testing.T) {
	tests := []test_helper.TestItem{
		{``, `comp/assign/type/a`},
	}
	test_helper.RunTest(t, "assignment_error_test.pf", tests, test_helper.TestInitializationErrors)
}

func TestBooleans(t *testing.T) {
	tests := []test_helper.TestItem{
		{`true : 5; else : 6`, `5`},
		{`false : 5; else : 6`, `6`},
		{`1 == 1 : 5; else : 6`, `5`},
		{`1 == 2 : 5; else : 6`, `6`},
		{`testNot true"`, `false`},
		{`testNot false"`, `true`},
		{`testNot 5"`, `vm/bool/not`},
		{`testOr true, false`, `true`},
		{`testOr false, false`, `false`},
		{`testOr true, true`, `true`},
		{`testOr true, false`, `true`},
		{`testOr 5, false`, `vm/bool/or/left`},
		{`testOr false, 5`, `vm/bool/or/right`},
		{`testAnd true, false`, `false`},
		{`testAnd false, false`, `false`},
		{`testAnd true, true`, `true`},
		{`testAnd true, false`, `false`},
		{`testAnd 5, true`, `vm/bool/and/left`},
		{`testAnd true, 5`, `vm/bool/and/right`},
		{`testConditional true`, `true`},
		{`testConditional false`, `false`},
		{`not true`, `false`},
		{`not false`, `true`},
		{`false and false`, `false`},
		{`true and false`, `false`},
		{`false and true`, `false`},
		{`true and true`, `true`},
		{`false or false`, `false`},
		{`true or false`, `true`},
		{`false or true`, `true`},
		{`true or true`, `true`},
	}
	test_helper.RunTest(t, "boolean_test.pf", tests, test_helper.TestValues)
}
func TestBooleanCompilerErrors(t *testing.T) {
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
func TestBuiltins(t *testing.T) {
	tests := []test_helper.TestItem{
		{`5.0 + 2.0`, `7`},
		{`5 + 2`, `7`},
		{`[1, 2] + [3, 4]`, `[1, 2, 3, 4]`},
		{`set(1, 2) + set(3, 4) == set(1, 2, 3, 4)`, `true`},
		{`'h' + 'i'`, `"hi"`},
		{`'j' + "ello"`, `"jello"`},
		{`"jell" + 'o'`, `"jello"`},
		{`"jel" + "lo"`, `"jello"`},
		{`5.0 / 2.0`, `2.5`},
		{`5 / 2`, `2.5`},
		{`5 / 2.0`, `2.5`},
		{`5.0 / 2`, `2.5`},
		{`5.0 > 2.0`, `true`},
		{`5.0 >= 2.0`, `true`},
		{`5 > 2`, `true`},
		{`5 >= 2`, `true`},
		{`5.0 < 2.0`, `false`},
		{`5.0 <= 2.0`, `false`},
		{`5 < 2`, `false`},
		{`5 <= 2`, `false`},
		{`"foo"::2`, `"foo"::2`},
		{`5 mod 2`, `1`},
		{`5.0 * 2.0`, `10`},
		{`5.0 * 2`, `10`},
		{`5 * 2.0`, `10`},
		{`5 * 2`, `10`},
		{`-5.0`, `-5`},
		{`-5`, `-5`},
		{`5.0 - 2.0`, `3`},
		{`5 - 2`, `3`},
		{`int/string`, `int/string`},
		{`[1, 2, 3] ...`, `(1, 2, 3)`},
		{`codepoint 'A'`, `65`},
		{`first (tuple 1, 2, 3, 4, 5)`, `1`},
		{`float 5`, `5`},
		{`float "5"`, `5`},
		{`5 in [1, 2, 3]`, `false`},
		{`5 in [1, 2, 3, 4, 5]`, `true`},
		{`5 in set 1, 2, 3`, `false`},
		{`5 in set 1, 2, 3, 4, 5`, `true`},
		{`5 in tuple 1, 2, 3`, `false`},
		{`5 in tuple 1, 2, 3, 4, 5`, `true`},
		{`5 in string`, `false`},
		{`5 in struct`, `false`},
		{`5 in int`, `true`},
		{`5 in int?`, `true`},
		{`int 5.2`, `5`},
		{`int "5"`, `5`},
		{`last (tuple 1, 2, 3, 4, 5)`, `5`},
		{`len [1, 2, 3]`, `3`},
		{`len (map "a"::1, "b"::2, "c"::3)`, `3`},
		{`len set 1, 2, 3`, `3`},
		{`len "Angela"`, `6`},
		{`len tuple 1, 2, 3`, `3`},
		{`literal 3`, `"3"`},
		{`literal "foo"`, `"\"foo\""`},
		{`literal 'q'`, `"'q'"`},
		{`rune 65`, `'A'`},
		{`map "a"::1, "b"::2`, `map("a"::1, "b"::2)`},
		{`set 1, 2, 3`, `set(1, 2, 3)`},
		{`string 4.0`, `"4"`},
		{`string 4`, `"4"`},
		{`tuple 1`, `tuple(1)`},
		{`type true`, `bool`},
		{`type bool`, `type`},
		{`[1, 2, 3] & 4`, `[1, 2, 3, 4]`},
		{`4 in (set(1, 2, 3) & 4)`, `true`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}
func TestCast(t *testing.T) {
	tests := []test_helper.TestItem{
		{`cast "foo", string`, `"foo"`},
		{`cast Uid(8), int`, `8`},
		{`cast 8, Uid`, `Uid(8)`},
		{`cast 0, Color`, `RED`},
		{`cast ["John", 22], Person`, `Person with (name::"John", age::22)`},
	}
	test_helper.RunTest(t, "cast_test.pf", tests, test_helper.TestValues)
}
func TestClones(t *testing.T) {
	tests := []test_helper.TestItem{
		{`FloatClone(4.2) == FloatClone(4.2)`, `true`},
		{`FloatClone(4.2) == FloatClone(9.9)`, `false`},
		{`IntClone(42) == IntClone(42)`, `true`},
		{`IntClone(42) == IntClone(99)`, `false`},
		{`ListClone([1, 2]) == ListClone[1, 2]`, `true`},
		{`ListClone([1, 2]) == ListClone([1, 3])`, `false`},
		{`ListClone([1, 2]) == ListClone([1, 2, 3])`, `false`},
		{`MapClone(map(1::2, 3::4)) == MapClone(3::4, 1::2)`, `true`},
		{`MapClone(map(1::2, 3::4)) == MapClone(map(1::2, 3::5))`, `false`},
		{`MapClone(map(1::2, 3::4)) == MapClone(map(1::2, 3::4, 5::6))`, `false`},
		{`PairClone(1::2) == PairClone(1::2)`, `true`},
		{`PairClone(1::2) == PairClone(2::2)`, `false`},
		{`PairClone(1::2) == PairClone(1::1)`, `false`},
		{`RuneClone('a') == RuneClone('a')`, `true`},
		{`RuneClone('a') == RuneClone('z')`, `false`},
		{`SetClone(set(1, 2)) == SetClone(2, 1)`, `true`},
		{`SetClone(set(1, 2)) == SetClone(set(1, 3))`, `false`},
		{`SetClone(set(1, 2)) == SetClone(set(1, 2, 3))`, `false`},
		{`StringClone("aardvark") == StringClone("aardvark")`, `true`},
		{`StringClone("aardvark") == StringClone("zebra")`, `false`},
		{`5 apples + 3 apples`, `apples(8)`},
		{`clones{list}`, `clones{list}`},
	}
	test_helper.RunTest(t, "clone_test.pf", tests, test_helper.TestValues)
}
func TestCorners(t *testing.T) {
	tests := []test_helper.TestItem{
		{`boo x`, `(1, 2, 3)`},
		{`foo 1, 2`, `3`},
		{`moo 1, 2`, `3`},
	}
	test_helper.RunTest(t, "corners_test.pf", tests, test_helper.TestValues)
}
func TestEquality(t *testing.T) {
	tests := []test_helper.TestItem{
		{`5.0 == 2.0`, `false`},
		{`5.0 != 2.0`, `true`},
		{`5 == 2`, `false`},
		{`5 != 2`, `true`},
		{`true != false`, `true`},
		{`"foo" == "foo"`, `true`},
		{`int == int`, `true`},
		{`struct == struct`, `true`},
		{`[1, 2, 3] == [1, 2, 3]`, `true`},
		{`[1, 2, 4] == [1, 2, 3]`, `false`},
		{`[1, 2, 3, 4] == [1, 2, 3]`, `false`},
		{`[1, 2, 3] == [1, 2, 3, 4]`, `false`},
		{`set(1, 2, 3) == set(1, 2, 3)`, `true`},
		{`set(1, 2, 4) == set(1, 2, 3)`, `false`},
		{`set(1, 2, 3, 4) == set(1, 2, 3)`, `false`},
		{`set(1, 2, 3) == set(1, 2, 3, 4)`, `false`},
		{`1::2 == 1::2`, `true`},
		{`1::2 == 2::2`, `false`},
		{`1::2 == 1::1`, `false`},
		{`map(1::2, 3::4) == map(1::2, 3::4)`, `true`},
		{`map(1::2, 3::4) == map(1::2, 4::4)`, `false`},
		{`map(1::2, 3::4) == map(1::2, 3::5)`, `false`},
		{`map(1::2, 3::4) == map(1::2, 3::4, 5::6)`, `false`},
		{`map(1::2, 3::4, 5::6) == map(1::2, 3::4)`, `false`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}
func TestEqualityCompilerErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`(error "foo") == 42`, `comp/error/eq/a`},
		{`42 == (error "foo")`, `comp/error/eq/b`},
		{`42 == "foo"`, `comp/eq/types`},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestExternals(t *testing.T) {
	tests := []test_helper.TestItem{
		{`zort.square 5`, `25`},
		{`type zort.Color`, `type`},
		{`zort.RED`, `zort.RED`},
		{`type zort.RED`, `zort.Color`},
		{`zort.RED in zort.Color`, `true`},
		{`zort.Color(4)`, `zort.BLUE`},
		{`zort.Person "John", 22`, `zort.Person with (name::"John", age::22)`},
		{`zort.Tone LIGHT, BLUE`, `zort.Tone with (shade::zort.LIGHT, color::zort.BLUE)`},
		{`zort.Qux 5`, `zort.Qux(5)`},
		{`zort.blerp`, `"Blerp"`},
		{`zort.spong()`, `"Spong"`},
		{`zort.moo boo 8`, `"Moo boo"`},
		{`zort.moo boo coo 8`, `"Moo boo coo"`},
		{`zort.moo zoo`, `"Moo zoo"`},
		{`zort.xuq 9 mip`, `"Xuq _ mip"`},
		{`8 zort.spoit`, `"_ spoit"`},
		{`8 zort.qux 9`, `"_ qux _"`},
		{`8 zort.bing 9 bong`, `"_ bing _ bong"`},
	}
	test_helper.RunTest(t, "external_test.pf", tests, test_helper.TestValues)
}
func TestFancyFunctions(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 99`, `"foo _"`},
		{`spong()`, `"spong _"`},
		{`blerp`, `"blerp"`},
		{`moo boo 8`, `"moo boo _"`},
		{`moo boo coo 8`, `"moo boo coo _"`},
		{`moo zoo`, `"moo zoo"`},
		{`9 spoit`, `"_ spoit"`},
		{`xuq 9 mip`, `"xuq _ mip"`},
		{`troz 8 nerf 9`, `"troz _ nerf _"`},
		{`goo 8 hoo 9 spoo 0`, `"goo _ hoo _ spoo _"`},
		{`gee 8 hee 9 spee`, `"gee _ hee _ spee"`},
		{`gah 8 hah 9 spah blah`, `"gah _ hah _ spah blah"`},
		{`8 bing 9 bong`, `"_ bing _ bong"`},
		{`8 ding 9 dong 0 dang`, `"_ ding _ dong _ dang"`},
	}
	test_helper.RunTest(t, "fancy_function_test.pf", tests, test_helper.TestValues)
}
func TestForLoops(t *testing.T) {
	tests := []test_helper.TestItem{
		{`fib 8`, `21`},
		{`collatzA 42`, `1`},
		{`collatzB 42`, `1`},
		{`evens Color`, `[RED, YELLOW, BLUE]`},
		{`evens "Angela"`, `['A', 'g', 'l']`},
		{`evens myList`, `[PURPLE, GREEN, ORANGE]`},
		{`find GREEN, Color`, `3`},
		{`find GREEN, myList`, `2`},
		{`find GREEN, myMap`, `"c"`},
		{`allKeys myList`, `[0, 1, 2, 3, 4, 5]`},
		{`allKeys "Angela"`, `[0, 1, 2, 3, 4, 5]`},
		{`allValues myList`, `[PURPLE, BLUE, GREEN, YELLOW, ORANGE, RED]`},
		{`showRange 3, 8`, `[0::3, 1::4, 2::5, 3::6, 4::7]`},
		{`showRangeKeys 3, 8`, `[0, 1, 2, 3, 4]`},
		{`showRangeValues 3, 8`, `[3, 4, 5, 6, 7]`},
		{`showRange 8, 3`, `[0::7, 1::6, 2::5, 3::4, 4::3]`},
		{`showRangeKeys 8, 3`, `[0, 1, 2, 3, 4]`},
		{`showRangeValues 8, 3 `, `[7, 6, 5, 4, 3]`},
		{`x`, `10`},
	}
	test_helper.RunTest(t, "for_loop_test.pf", tests, test_helper.TestValues)
}
func TestFunctionOverloading(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 42`, `"int"`},
		{`foo "zort"`, `"string"`},
		{`foo 42, true`, `"any?, bool"`},
		{`foo 42.0, true`, `"any?, bool"`},
		{`foo true, true`, `"bool, bool"`},
	}
	test_helper.RunTest(t, "overloading_test.pf", tests, test_helper.TestValues)
}
func TestFunctionSharing(t *testing.T) {
	tests := []test_helper.TestItem{
		{`C(1, 2) in Addable`, `true`},
		{`C(1, 2) in summer.Addable`, `true`},
		{`C(1, 2) in summer.Rotatable`, `true`},
		{`summer.sum [C(1, 2), C(3, 4), C(5, 6)]`, `C with (real::9, imaginary::12)`},
		{`summer.rotAll [C(1, 2), C(3, 4)]`, `[C with (real::-2, imaginary::1), C with (real::-4, imaginary::3)]`},
	}
	test_helper.RunTest(t, "function_sharing_test.pf", tests, test_helper.TestValues)
}
func TestFunctionSyntaxCalls(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo "bing"`, `"foo bing"`},
		{`"bing" zort`, `"bing zort"`},
		{`"bing" troz "bong"`, `"bing troz bong"`},
		{`moo "bing" goo`, `"moo bing goo"`},
		{`flerp "bing" blerp "bong"`, `"flerp bing blerp bong"`},
		{`qux`, `"qux"`},
	}
	test_helper.RunTest(t, "function_call_test.pf", tests, test_helper.TestValues)
}
func TestGocode(t *testing.T) {
	// no t.Parallel()
	tests := []test_helper.TestItem{
		{`anyTest 42`, `42`},
		{`variadicAnyTest 2, 42, true, "foo", 9.9`, `"foo"`},
		{`boolTest true`, `false`},
		{`float 4.2`, `4.2`},
		{`intTest 42`, `84`},
		{`listTest([1, 2])`, `[1, 2]`},
		{`mapTest(map(1::2, 3::4)) == map(1::2, 3::4)`, `true`},
		{`pairTest(1::2) == 1::2`, `true`},
		{`runeTest('q') == 'q'`, `true`},
		{`setTest(set(1, 2)) == set(1, 2)`, `true`},
		{`stringTest "aardvark"`, `"aardvark"`},
		{`tupleTest(tuple(1, 2)) == [1, 2]`, `true`},
		{`variadicTest(2, "fee", "fie", "fo", "fum") == "fo"`, `true`},
		{`enumTest BLUE`, `BLUE`},
		{`intCloneTest IntClone(5)`, `IntClone(5)`},
		{`constructPerson "Doug", 42`, `Person with (name::"Doug", age::42)`},
		{`deconstructPerson Person "Doug", 42`, `("Doug", 42)`},
		{`floatCloneTest(FloatClone(4.2)) == FloatClone(4.2)`, `true`},
		{`intCloneTest(IntClone(42)) == IntClone(42)`, `true`},
		{`listCloneTest(ListClone([1, 2])) == ListClone([1, 2])`, `true`},
		{`mapCloneTest(MapClone(map(1::2, 3::4))) == MapClone(map(1::2, 3::4))`, `true`},
		{`pairCloneTest(PairClone(1::2)) == PairClone(1::2)`, `true`},
		{`runeCloneTest(RuneClone('q')) == RuneClone('q')`, `true`},
		{`setCloneTest(SetClone(set(1, 2))) == SetClone(set(1, 2))`, `true`},
		{`stringCloneTest(StringClone("zort")) == StringClone("zort")`, `true`},
		{`commandTest`, `OK`},
		{`applyFunction(2, (func(i int) : 2 * i))`, `4`},
		{`type(multiplyBy(3))`, `func`},
		{`getType(3)`, `func`},
		{`multiply 2, 3`, `6`},
	}
	test_helper.RunTest(t, "gocode_test.pf", tests, test_helper.TestValues)
	test_helper.Teardown("gocode_test.pf")
}
func TestHighlighter(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Type`, `[38;2;78;201;176mType[0m`},
		{`int`, `[38;2;78;201;176mint[0m`},
		{`int // comment`, `[38;2;78;201;176mint[0m [38;2;106;153;85m// comment[0m`},
		{`"string"`, `[38;2;206;145;120m"string"[0m`},
		{`42`, `[38;2;181;206;168m42[0m`},
		{`ENUM`, `[38;2;79;193;255mENUM[0m`},
		{`~~ docstring`, `[38;2;244;71;71m[4m~~[0m docstring`},
		{`true`, `[38;2;86;156;214mtrue[0m`},
		{`'q'`, `[38;2;206;145;120m'q'[0m`},
		{`else`, `[38;2;197;134;192melse[0m`},
		{"`foo`", "[38;2;206;145;120m`foo`[0m"},
		{`0b10`, `[38;2;181;206;168m0b10[0m`},
		{`0o10`, `[38;2;181;206;168m0o10[0m`},
		{`0x10`, `[38;2;181;206;168m0x10[0m`},
		{`foo(bar(spong()))`, `foo[38;2;255;215;0m([0mbar[38;2;218;112;214m([0mspong[38;2;23;159;255m([0m[38;2;23;159;255m)[0m[38;2;218;112;214m)[0m[38;2;255;215;0m)[0m`},
		{`(]`, `[38;2;255;215;0m([0m[38;2;244;71;71m[4m][0m`},
		{`int?`, `[38;2;78;201;176mint?[0m`},
		{`int!`, `[38;2;78;201;176mint![0m`},
		{`.`, `[38;2;86;156;214m.[0m`},
	}
	test_helper.RunTest(t, "highlighter_test.pf", tests, test_helper.TestHighlighter)
}
func TestImports(t *testing.T) {
	tests := []test_helper.TestItem{
		{`qux.square 5`, `25`},
		{`type qux.Color`, `type`},
		{`qux.RED`, `qux.RED`},
		{`type qux.RED`, `qux.Color`},
		{`qux.RED in qux.Color`, `true`},
		{`qux.Color(4)`, `qux.BLUE`},
		{`qux.Person "John", 22`, `qux.Person with (name::"John", age::22)`},
		{`qux.Tone LIGHT, BLUE`, `qux.Tone with (shade::qux.LIGHT, color::qux.BLUE)`},
		{`troz.sumOfSquares 3, 4`, `25`},
	}
	test_helper.RunTest(t, "import_test.pf", tests, test_helper.TestValues)
}
func TestIndexing(t *testing.T) {
	tests := []test_helper.TestItem{
		{`DARK_BLUE[shade]`, `DARK`},
		{`myColor[shade]`, `LIGHT`},
		{`DARK_BLUE[KEY]`, `DARK`},
		{`myColor[KEY]`, `LIGHT`},
		{`DARK_BLUE[key]`, `DARK`},
		{`myColor[key]`, `LIGHT`},
		{`"Angela"[3]`, `'e'`},
		{`"Angela"[2::5]`, `"gel"`},
		{`myWord[2::5]`, `"gel"`},
		{`myList[2]`, `[5, 6]`},
		{`myList[myNumber]`, `[5, 6]`},
		{`myList[0::2]`, `[[1, 2], [3, 4]]`},
		{`myList[myIntPair]`, `[[1, 2], [3, 4]]`},
		{`("a", "b", "c", "d")[2]`, `"c"`},
		{`("a", "b", "c", "d")[myIntPair]`, `("a", "b")`},
		{`"Angela"[myIntPair]`, `"An"`},
		{`myWord[myIntPair]`, `"An"`},
		{`myPair[0]`, `"foo"`},
		{`myMap["a"]`, `[1, 2]`},
		{`foo myMap, myIndex`, `[1, 2]`},
		{`foo myList, myNumber`, `[5, 6]`},
		{`foo myColor, key`, `LIGHT`},
		{`foo myPair, myOtherNumber`, `"bar"`},
		{`foo myWord, myNumber`, `'g'`},
	}
	test_helper.RunTest(t, "index_test.pf", tests, test_helper.TestValues)
}
func TestIndexingCompilerErrors(t *testing.T) {
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
func TestImperative(t *testing.T) {
	tests := []test_helper.TestItem{
		{`zort false`, `7`},
	}
	test_helper.RunTest(t, "imperative_test.pf", tests, test_helper.TestOutput)
}
func TestInnerFunctionsAndVariables(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 42`, `42`},
		{`zort 3, 5`, `(25, 15)`},
		{`troz 2`, `2200`},
	}
	test_helper.RunTest(t, "inner_test.pf", tests, test_helper.TestValues)
}
func TestInterfaces(t *testing.T) {
	tests := []test_helper.TestItem{
		{`BLERP in Addable`, `true`},
		{`Fnug(5) in Addable`, `true`},
		{`ZORT in Foobarable`, `true`},
		{`true in Addable`, `false`},
		{`Fnug(5) in Foobarable`, `false`},
	}
	test_helper.RunTest(t, "interface_test.pf", tests, test_helper.TestValues)
}
func TestLambdas(t *testing.T) {
	tests := []test_helper.TestItem{
		{`apply DOUBLE, 42`, `84`},
		{`apply "DOUBLE", 42`, `vm/apply/func`},
	}
	test_helper.RunTest(t, "lambda_test.pf", tests, test_helper.TestValues)
}
func TestLiterals(t *testing.T) {
	tests := []test_helper.TestItem{
		{`"foo"`, `"foo"`},
		{"`foo`", `"foo"`},
		{`'q'`, `'q'`},
		{`true`, `true`},
		{`false`, `false`},
		{`42.0`, `42`},
		{`42`, `42`},
		{`0b101010`, `42`},
		{`0o52`, `42`},
		{`0x2A`, `42`},
		{`NULL`, `NULL`},
		{`OK`, `OK`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}
func TestLogging(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 8`, test_helper.Foo8Result},
	}
	test_helper.RunTest(t, "logging_test.pf", tests, test_helper.TestOutput)
}
func TestMiscellaneousCompilerErrors(t *testing.T) {
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
		{`-- foo |(1, 2, 3)| bar`, `comp/snippet/tuple`},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestParameterizedTypes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Z{5}(3) + Z{5}(4)`, `Z{5}(2)`},
		{`Zort{0}(0::0)`, `Zort{0}(0::0)`},
		{`Troz{0}(0)`, `Troz{0}(0)`},
	}
	test_helper.RunTest(t, "parameterized_type_test.pf", tests, test_helper.TestValues)
}

func TestPiping(t *testing.T) {
	tests := []test_helper.TestItem{
		{`["fee", "fie", "fo", "fum"] -> len`, `4`},
		{`["fee", "fie", "fo", "fum"] >> len`, `[3, 3, 2, 3]`},
		{`["fee", "fie", "fo", "fum"] -> that + ["foo"]`, `["fee", "fie", "fo", "fum", "foo"]`},
		{`["fee", "fie", "fo", "fum"] >> that + "!"`, `["fee!", "fie!", "fo!", "fum!"]`},
		{`[1, 2, 3, 4] ?> that mod 2 == 0`, `[2, 4]`},
		{`ks >> MP[that]`, `["fee", "fie", "fo", "fum"]`},
		{`foo ks`, `["fee", "fie", "fo", "fum"]`},
		{`goo ks`, `["fee", "fie", "fo", "fum"]`},
		{`[1, 2, 3] >> double`, `[2, 4, 6]`},
		{`boo([1, 2, 3], double)`, `[2, 4, 6]`},
	}
	test_helper.RunTest(t, "piping_test.pf", tests, test_helper.TestValues)
}
func TestRecursion(t *testing.T) {
	tests := []test_helper.TestItem{
		{`fac 5`, `120`},
		{`power 3, 4`, `81`},
		{`inFac 5`, `120`},
	}
	test_helper.RunTest(t, "recursion_test.pf", tests, test_helper.TestValues)
}
func TestRecursion2(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo [1, 2, 3, 4]`, `10`},
		{`chunk "f(((o))o)"`, `['f', [[['o']], 'o']]`},
	}
	test_helper.RunTest(t, "recursion_test_2.pf", tests, test_helper.TestValues)
}
func TestReflection(t *testing.T) {
	tests := []test_helper.TestItem{
		{`reflect.isStruct Varchar{8}`, `false`},
	}
	test_helper.RunTest(t, "reflect_test.pf", tests, test_helper.TestValues)
}
func TestRef(t *testing.T) {
	tests := []test_helper.TestItem{
		{`x ++`, `OK`},
	}
	test_helper.RunTest(t, "ref_test.pf", tests, test_helper.TestValues)
}
func TestSnippets(t *testing.T) {
	tests := []test_helper.TestItem{
		{`(qux 5)[0]`, `"foo "`},
		{`(qux 5)[1]`, `10`},
		{`(qux 5)[2]`, `" bar"`},
	}
	test_helper.RunTest(t, "snippets_test.pf", tests, test_helper.TestValues)
}
func TestStructs(t *testing.T) {
	tests := []test_helper.TestItem{
		{`doug`, `Person with (name::"Douglas", age::42)`},
		{`tom in Cat`, `true`},
	}
	test_helper.RunTest(t, "struct_test.pf", tests, test_helper.TestValues)
}
func TestTry(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 3`, `4`},
		{`foo 0`, `"Oops"`},
	}
	test_helper.RunTest(t, "try_test.pf", tests, test_helper.TestOutput)
}
func TestTuples(t *testing.T) {
	tests := []test_helper.TestItem{
		{`(1, 2), 3`, `(1, 2, 3)`},
		{`1, (2, 3)`, `(1, 2, 3)`},
		{`(1, 2), (3, 4)`, `(1, 2, 3, 4)`},
		{`()`, `()`},
		{`type tuple "foo", "bar"`, `tuple`},
		{`len tuple "foo", "bar"`, `2`},
		{`1 in tuple(1, 2)`, `true`},
		{`string(X)`, `"(2, 3)"`},
		{`len tuple 1, X`, `3`},
		{`len tuple X, 1`, `3`},
		{`len tuple W, Z`, `8`},
		{`foo 1, X`, `3`},
		{`foo X, 1`, `3`},
		{`foo W, Z`, `8`},
	}
	test_helper.RunTest(t, "tuples_test.pf", tests, test_helper.TestValues)
}
func TestTypes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Color(4)`, `BLUE`},
		{`DARK_BLUE`, `Tone with (shade::DARK, color::BLUE)`},
		{`type DARK_BLUE`, `Tone`},
		{`type RED`, `Color`},
		{`keys DARK_BLUE`, `[shade, color]`},
		{`DARK_BLUE[shade]`, `DARK`},
		{`DARK_BLUE[color]`, `BLUE`},
		{`GREEN == GREEN`, `true`},
		{`GREEN == ORANGE`, `false`},
		{`GREEN != GREEN`, `false`},
		{`GREEN != ORANGE`, `true`},
		{`PURPLE in MyType`, `true`},
		{`Tone/Shade/Color`, `MyType`},
		{`Tone(LIGHT, GREEN)`, `Tone with (shade::LIGHT, color::GREEN)`},
		{`Tone(LIGHT, GREEN) == DARK_BLUE`, `false`},
		{`Tone(LIGHT, GREEN) != DARK_BLUE`, `true`},
		{`troz DARK_BLUE`, `Tone with (shade::DARK, color::BLUE)`},
		{`foo 3, 5`, `8`},
		{`Tone with (shade::LIGHT, color::RED)`, `Tone with (shade::LIGHT, color::RED)`},
	}
	test_helper.RunTest(t, "user_types_test.pf", tests, test_helper.TestValues)
}
func TestTypeAccessErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Pair 1, 2`, `comp/private`},
		{`Suit`, `comp/private/type`},
		{`HEARTS`, `comp/ident/private`},
		{`one`, `comp/ident/private`},
	}
	test_helper.RunTest(t, "user_types_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestTypeExpressionCompilerErrors(t *testing.T) {
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
func TestTypeInstances(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Z{3}(2) in Z{3}`, `true`},
	}
	test_helper.RunTest(t, "type_instances_test.pf", tests, test_helper.TestValues)
}
func TestValid(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 3`, `4`},
		{`foo 0`, `Error`},
	}
	test_helper.RunTest(t, "valid_test.pf", tests, test_helper.TestValues)
}
func TestVariablesAndConsts(t *testing.T) {
	tests := []test_helper.TestItem{
		{`A`, `42`},
		{`getB`, `99`},
		{`changeZ`, `OK`},
		{`v`, `true`},
		{`w`, `42`},
		{`y = NULL`, "OK"},
	}
	test_helper.RunTest(t, "variables_test.pf", tests, test_helper.TestValues)
}
func TestVariableAccessErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`B`, `comp/ident/private`},
		{`A = 43`, `comp/assign/const`},
		{`z`, `comp/ident/private`},
		{`secretB`, `comp/private`},
		{`secretZ`, `comp/private`},
	}
	test_helper.RunTest(t, "variables_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestVariableCompilerErrors(t *testing.T) {
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
func TestWith(t *testing.T) {
	tests := []test_helper.TestItem{
		{`john with name::"Susan", age::23`, `Person with (name::"Susan", age::23)`},
		{`john with age::23`, `Person with (name::"John", age::23)`},
		{`myList with diffList`, `["x", "y", "c", "d"]`},
		{`myMap with "a"::99`, `map("a"::99, "b"::2, "c"::3, "d"::4)`},
		{`myMap with "z"::42`, `map("a"::1, "b"::2, "c"::3, "d"::4, "z"::42)`},
		{`myMap with diffMap`, `map("a"::99, "b"::99, "c"::3, "d"::4)`},
		{`otherMap with ["a", 1]::99`, `map("a"::[0, 99], "b"::[2, 3])`},
		{`myMap with "a"::99, "z"::42`, `map("a"::99, "b"::2, "c"::3, "d"::4, "z"::42)`},
		{`myMap without "a"`, `map("b"::2, "c"::3, "d"::4)`},
		{`myMap without "a", "b"`, `map("c"::3, "d"::4)`},
	}
	test_helper.RunTest(t, "with_test.pf", tests, test_helper.TestValues)
}
func TestWrappers(t *testing.T) {
	// no t.Parallel()
	tests := []test_helper.TestItem{
		{`Uint_32(5) == Uint_32(6)`, `false`},
		{`Uint_32(5) == Uint_32(5)`, `true`},
		{`Uint_32(5)`, `Uint_32(5)`},
		{`literal Uint_32(5)`, `"Uint_32(5)"`},
	}
	test_helper.RunTest(t, "wrapper_test.pf", tests, test_helper.TestValues)
	test_helper.Teardown("wrapper_test.pf")
}
