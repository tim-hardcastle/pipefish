package vm_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tim-hardcastle/pipefish/source/test_helper"
	"github.com/tim-hardcastle/pipefish/source/text"
)

func TestAssignment(t *testing.T) {
	tests := []test_helper.TestItem{
		{`x`, `'q'`},
		{`y`, `2`},
		{`x rune, y int = 'z', 42`, `OK`},
		{`y = 42`, `OK`},
	}
	test_helper.RunTest(t, "assignment_test.pf", tests, test_helper.TestValues)
}

func TestBooleans(t *testing.T) {
	tests := []test_helper.TestItem{
		{`F or T`, `true`},
		{`T or F`, `true`},
		{`T or T`, `true`},
		{`F or F`, `false`},
		{`T and F`, `false`},
		{`F and T`, `false`},
		{`F and F`, `false`},
		{`T and T`, `true`},
		{`T : 5`, `5`},
		{`not T`, `false`},
		{`not F`, `true`},
		{`Q or T`, `vm/bool/or/left`},
		{`F or Q`, `vm/bool/or/right`},
		{`Q and F`, `vm/bool/and/left`},
		{`T and Q`, `vm/bool/and/right`},
		{`Q : 5`, `vm/bool/cond`},
		{`not Q`, `vm/bool/not`},
	}
	test_helper.RunTest(t, "boolean_errors_test.pf", tests, test_helper.TestValues)
}

func TestBuiltins(t *testing.T) {
	tests := []test_helper.TestItem{
		{`5.0 + 2.0`, `7.0`},
		{`5.0 + 2.5`, `7.5`},
		{`5 + 2`, `7`},
		{`[1, 2] + [3, 4]`, `[1, 2, 3, 4]`},
		{`set(1, 2) + set(3, 4) == set(1, 2, 3, 4)`, `true`},
		{`'j' & "ello"`, `"jello"`},
		{`"jell" & 'o'`, `"jello"`},
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
		{`5.0 * 2.0`, `10.0`},
		{`5.0 * 2`, `10.0`},
		{`5 * 2.0`, `10.0`},
		{`5 * 2`, `10`},
		{`-5.0`, `-5.0`},
		{`-5`, `-5`},
		{`5.0 - 2.0`, `3.0`},
		{`5 - 2`, `3`},
		{`int/string`, `int/string`},
		{`[1, 2, 3] ...`, `(1, 2, 3)`},
		{`codepoint 'A'`, `65`},
		{`first (tuple 1, 2, 3, 4, 5)`, `1`},
		{`float 5`, `5.0`},
		{`float "5"`, `5.0`},
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
		{`len keys (map "a"::1, "b"::2, "c"::3)`, `3`},
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
		{`set(1, 2, 3) /\ set(2, 3, 4) == set(2, 3)`, `true`},
		{`set(1, 2, 3) - set(3, 4) == set(1, 2)`, `true`},
		{`string 4.0`, `"4.0"`},
		{`string 4`, `"4"`},
		{`tuple 1`, `tuple(1)`},
		{`type true`, `bool`},
		{`type bool`, `type`},
		{`[1, 2, 3] & 4`, `[1, 2, 3, 4]`},
		{`4 in (set(1, 2, 3) & 4)`, `true`},
		{`7 / zero`, `vm/div/zero/a`},
		{`7.0 / floatZero`, `vm/div/zero/b`},
		{`7 div zero`, `vm/div/zero/c`},
		{`7.0 / zero`, `vm/div/zero/d`},
		{`7 / floatZero`, `vm/div/zero/e`},
		{`7 mod zero`, `vm/mod/zero`},
		{`map (badKey::2)`, `vm/map/key`},
	}
	test_helper.RunTest(t, "builtins_test.pf", tests, test_helper.TestValues)
}
func TestCast(t *testing.T) {
	tests := []test_helper.TestItem{
		{`cast string, "foo"`, `"foo"`},
		{`cast int, Uid(8)`, `8`},
		{`cast Uid, 8`, `Uid(8)`},
		{`cast Color, 0`, `RED`},
		{`cast Person, ["John", 22]`, `Person("John", 22)`},
		{`cast castTo, "foo"`, `vm/cast/concrete`},
		{`cast Person, castThing`, `vm/cast`},
		{`cast Color, loNum`, `vm/cast/enum`},
		{`cast Color, hiNum`, `vm/cast/enum`},
		{`cast Person, badFields`, `vm/cast/fields`},
		{`cast Person, badTypes`, `vm/cast/types`},
		{`float castThing`, `vm/string/float`},
		{`int castThing`, `vm/string/int`},
	}
	test_helper.RunTest(t, "cast_test.pf", tests, test_helper.TestValues)
}
func TestClones(t *testing.T) {
	tests := []test_helper.TestItem{
		{`FloatClone(4.2) == FloatClone(4.2)`, `true`},
		{`FloatClone(4.2) == FloatClone(9.9)`, `false`},
		{`IntClone(42) == IntClone(42)`, `true`},
		{`IntClone(42) == IntClone(99)`, `false`},
		{`ListClone([1, 2]) == ListClone([1, 2])`, `true`},
		{`ListClone([1, 2]) == ListClone([1, 3])`, `false`},
		{`ListClone([1, 2]) == ListClone([1, 2, 3])`, `false`},
		{`MapClone(map(1::2, 3::4)) == MapClone(map(3::4, 1::2))`, `true`},
		{`MapClone(map(1::2, 3::4)) == MapClone(map(1::2, 3::5))`, `false`},
		{`MapClone(map(1::2, 3::4)) == MapClone(map(1::2, 3::4, 5::6))`, `false`},
		{`PairClone(1::2) == PairClone(1::2)`, `true`},
		{`PairClone(1::2) == PairClone(2::2)`, `false`},
		{`PairClone(1::2) == PairClone(1::1)`, `false`},
		{`RuneClone('a') == RuneClone('a')`, `true`},
		{`RuneClone('a') == RuneClone('z')`, `false`},
		{`SetClone(set(1, 2)) == SetClone(set(1, 2))`, `true`},
		{`SetClone(set(1, 2)) == SetClone(set(1, 3))`, `false`},
		{`SetClone(set(1, 2)) == SetClone(set(1, 2, 3))`, `false`},
		{`StringClone("aardvark") == StringClone("aardvark")`, `true`},
		{`StringClone("aardvark") == StringClone("zebra")`, `false`},
		{`5 apples + 3 apples`, `apples(8)`},
		{`clones{list}`, `clones{list}`},
		{`getClones number`, `vm/clones/type`},
	}
	test_helper.RunTest(t, "clone_test.pf", tests, test_helper.TestValues)
}

func TestConcatenation(t *testing.T) {
	tests := []test_helper.TestItem{
		{`conc false, true`, `(1, 2, 3)`},
		{`conc false, false`, `(1, 1)`},
		{`conc true, false`, `(2, 3, 1)`},
		{`conc true, true`, `(2, 3, 2, 3)`},
	}
	test_helper.RunTest(t, "concatenation_test.pf", tests, test_helper.TestValues)
}

func TestConditionals(t *testing.T) {
	tests := []test_helper.TestItem{
		{`true : 5; else : 6`, `5`},
		{`false : 5; else : 6`, `6`},
		{`1 == 1 : 5; else : 6`, `5`},
		{`1 == 2 : 5; else : 6`, `6`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}
func TestCorners(t *testing.T) {
	tests := []test_helper.TestItem{
		{`boo x`, `(1, 2, 3)`},
		{`foo 1, 2`, `3`},
		{`moo 1, 2`, `3`},
	}
	test_helper.RunTest(t, "corners_test.pf", tests, test_helper.TestValues)
}

func TestDump(t *testing.T) { // We want to make sure that if the service is broken, queries get handed off to the empty service.
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/dump.pf"`, `Starting script [36m"dump.pf"[39m as service [36m"dump"[39m.`},
		{`hub dump "big"`, "# Function dump of `big`\n\n## Code dump for function `big` with sig int\n\n@71 : asgm m263 <- m261  // Assign to memory.\n@72 : gtei m262 <- m263 m265  // Int comparison with >=.\n@73 : asgm m266 <- m262  // Assign to memory.\n@74 : qtru m266 @77  // Test true.\n@75 : asgm m268 <- m267  // Assign to memory.\n@76 : jmp @78  // Jump.\n@77 : asgm m268 <- m3  // Assign to memory.\n@78 : qsat m268 @81  // Test not `UNSAT`.\n@79 : asgm m270 <- m268  // Assign to memory.\n@80 : jmp @82  // Jump.\n@81 : asgm m270 <- m269  // Assign to memory.\n@82 : ret  // Return."},
		{`hub dump m "big"`, "# Function dump of `big`\n\n## Code dump for function `big` with sig int\n\n@71 : asgm m263 <- m261  // Assign to memory.\n@72 : gtei m262 <- m263 m265  // Int comparison with >=.\n@73 : asgm m266 <- m262  // Assign to memory.\n@74 : qtru m266 @77  // Test true.\n@75 : asgm m268 <- m267  // Assign to memory.\n@76 : jmp @78  // Jump.\n@77 : asgm m268 <- m3  // Assign to memory.\n@78 : qsat m268 @81  // Test not `UNSAT`.\n@79 : asgm m270 <- m268  // Assign to memory.\n@80 : jmp @82  // Jump.\n@81 : asgm m270 <- m269  // Assign to memory.\n@82 : ret  // Return.\n\n### Memory dump for function `big` with sig int`\n\nm261 : UNDEFINED VALUE::UNDEFINED VALUE!\nm262 : error::\x1b[31mError\x1b[39m: something unexpected has gone wrong at line \x1b[33m4:6-8\x1b[39m of \x1b[36m\"../hub/test-files/dump.pf\"\x1b[39m. \nm263 : UNDEFINED VALUE::UNDEFINED VALUE!\nm264 : BLING::>=\nm265 : int::100\nm266 : UNDEFINED VALUE::UNDEFINED VALUE!\nm267 : string::\"big\"\nm268 : UNDEFINED VALUE::UNDEFINED VALUE!\nm269 : string::\"small\"\nm270 : UNDEFINED VALUE::UNDEFINED VALUE!"},
		{`hub halt "dump"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestEnums(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Color goodNum`, `BLUE`},
		{`Color hiNum`, `vm/enum`},
		{`int BLUE`, `2`},
	}
	test_helper.RunTest(t, "enums_test.pf", tests, test_helper.TestValues)
}
func TestEof(t *testing.T) {
	tests := []test_helper.TestItem{
		{`troz 42`, `42`},
		{`zort 42`, `42`},
	}
	test_helper.RunTest(t, "eof_test.pf", tests, test_helper.TestValues)
}
func TestEquality(t *testing.T) { // Most of this gets tested elsewhere as a by-product of testing everything else,
	tests := []test_helper.TestItem{
		{`comp true, false`, `false`},
		{`comp 0.5, 0.5`, `true`},
		{`OK == OK`, `true`},
		{`IOTA == IOTA`, `false`},
		{`name == age`, `false`},
		{`comp int, string`, `false`},
		{`ta == tb`, `false`},
		{`ta == tc`, `false`},
		{`(1, 2, 3) == (1, 2, 3)`, `true`},
		{`(1, 2, 3) == (1, 2, 4)`, `false`},
		{`[1, 2, 3] == [1, 2, true]`, `false`},
		{`snippet(1, 2, 3) == snippet(1, 2, 3)`, `true`},
		{`snippet(1, 2, 3) == snippet(1, 2)`, `false`},
		{`snippet(1, 2, 3) == snippet(1, 2, "foo")`, `false`},
		{`snippet(1, 2, 3) == snippet(1, 2, 4)`, `false`},
		{`comp(foo(one), foo(two))`, `vm/equals/type`},
		{`zort zero, one`, `vm/div/zero/c`},
		{`zort one, zero`, `vm/div/zero/c`},
	}
	test_helper.RunTest(t, "equality_test", tests, test_helper.TestValues)
}
func TestEval(t *testing.T) {
	tests := []test_helper.TestItem{
		{`eval "4"`, `4`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}
func TestExternals(t *testing.T) {
	tests := []test_helper.TestItem{
		{`zort.square 5`, `25`},
		{`type zort.Color`, `type`},
		{`zort.RED`, `zort.RED`},
		{`type zort.RED`, `zort.Color`},
		{`zort.RED in zort.Color`, `true`},
		{`zort.Color(4)`, `zort.BLUE`},
		{`zort.Person "John", 22`, `zort.Person("John", 22)`},
		{`zort.Tone LIGHT, BLUE`, `zort.Tone(zort.LIGHT, zort.BLUE)`},
		{`zort.Qux 5`, `zort.Qux(5)`},
	}
	test_helper.RunTest(t, "external_test.pf", tests, test_helper.TestValues)
}
func TestForLoopRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`bar five`, `vm/typecheck/bound/init`},
		{`foo four`, `vm/typecheck/bound/update`},
		{`zort three`, `vm/typecheck/index/init`},
		{`qux three`, `vm/typecheck/index/update`},
		{`rozt three`, `vm/types.a`},
		{`zrot three`, `vm/types.a`},
		{`merp three`, `vm/for/condition`},
		{`count anyType`, `vm/for/type/a`},
		{`count intType`, `vm/for/type/b`},
		{`count x`, `vm/for/type/c`},
	}
	test_helper.RunTest(t, "for_loop_rtes_test.pf", tests, test_helper.TestValues)
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
		{`find "bar", mySnippet`, `1`},
		{`findInTuple "c", myTuple`, `2`},
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
		{`triangle 4`, `10`},
		{`countTuple myOtherTuple`, `3`},
		{`countTupleV myOtherTuple`, `3`},
		{`countTupleKv myOtherTuple`, `3`},
		{`count myPoint`, `2`},
		{`count mySnippet`, `3`},
		{`count Color`, `6`},
		{`count myMap`, `3`},
		{`count myList`, `6`},
		{`count mySet`, `5`},
		{`countV myMap`, `3`},
		{`countV myPoint`, `2`},
		{`countV mySnippet`, `3`},
		{`countV Color`, `6`},
		{`countV "Angela"`, `6`},
		{`countV myList`, `6`},
		{`countKv myPoint`, `2`},
		{`countKv mySnippet`, `3`},
		{`countKv Color`, `6`},
		{`countKv mySet`, `5`},
		{`addTuple myOtherTuple`, `6`},
		{`add myPoint`, `3`},
		{`add myOtherSet`, `6`},
	}
	test_helper.RunTest(t, "for_loop_test.pf", tests, test_helper.TestValues)
}
func TestFunctionSharing(t *testing.T) {
	tests := []test_helper.TestItem{
		{`C(1, 2) in Addable`, `true`},
		{`C(1, 2) in summer.Addable`, `true`},
		{`C(1, 2) in summer.Rotatable`, `true`},
		{`summer.sum [C(1, 2), C(3, 4), C(5, 6)]`, `C(9, 12)`},
		{`summer.rotAll [C(1, 2), C(3, 4)]`, `[C(-2, 1), C(-4, 3)]`},
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
		{`foo p 7`, `"foo p"`},
		{`foo q 7`, `"foo q"`},
	}
	test_helper.RunTest(t, "function_call_test.pf", tests, test_helper.TestValues)
}
func TestGocode(t *testing.T) {
	// no t.Parallel()
	if runtime.GOOS == "windows" {
		return
	}
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
		{`constructPerson "Doug", 42`, `Person("Doug", 42)`},
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
func TestHardwiredOps(t *testing.T) {
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
		{`1, (2, 3)`, `(1, 2, 3)`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}

func TestHttp(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub http`, "\x1b[32mOK\x1b[0m"},
		{`hub run "../hub/test-files/server.pf"`, `Starting script [36m"server.pf"[39m as service [36m"server"[39m.`},
		{`hub run "../hub/test-files/client.pf"`, `Starting script [36m"client.pf"[39m as service [36m"client"[39m.`},
		{`twice 2`, "4"},
		{`hub halt "client"`, `OK`},
		{`hub halt "server"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestImperative(t *testing.T) {
	tests := []test_helper.TestItem{
		{`zort false`, `7`},
		{`zort true`, `6`},
		{`qux false`, `5`},
		{`qux true`, `6`},
	}
	test_helper.RunTest(t, "imperative_test.pf", tests, test_helper.TestOutput)
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
		{`foo myList, 0::1`, `[[1, 2]]`},
		{`foo myColor, key`, `LIGHT`},
		{`foo myPair, myOtherNumber`, `"bar"`},
		{`foo myWord, myNumber`, `'g'`},
		{`"Angela"[3]`, `'e'`},
		{`myTuple[myIntPair]`, `(1, 2)`},
		{`myTuple[myNumber]`, `3`},
		{`mySnippet[myNumber]`, `" troz "`},
		{`myWord[myNumber]`, `'g'`},
		{`goo myTuple, myIntPair`, `(1, 2)`},
		{`goo myTuple, myNumber`, `3`},
		{`foo mySnippet, myNumber`, `" troz "`},
		{`foo myWord, myIntPair`, `"An"`},
		{`foo myClist, myCint`, `BLUE`},
		{`foo myClist, myIntPair`, `Clist[RED, GREEN]`},
	}
	test_helper.RunTest(t, "index_test.pf", tests, test_helper.TestValues)
}
func TestIndexingRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`[RED, GREEN, BLUE][myBool::2]`, `vm/index/a`},
		{`[RED, GREEN, BLUE][2::myBool]`, `vm/index/b`},
		{`[RED, GREEN, BLUE][myNegative::2]`, `vm/slice/list/c`},
		{`[RED, GREEN, BLUE][three::2]`, `vm/slice/list/d`},
		{`[RED, GREEN, BLUE][0::bigNumber]`, `vm/slice/list/e`},
		{`"aardvark"[myBool::2]`, `vm/index/a`},
		{`"aardvark"[2::myBool]`, `vm/index/b`},
		{`"aardvark"[myNegative::2]`, `vm/slice/string/c`},
		{`"aardvark"[three::2]`, `vm/slice/string/d`},
		{`"aardvark"[0::bigNumber]`, `vm/slice/string/e`},
		{`(1, 2, 3)[myBool::2]`, `vm/index/a`},
		{`(1, 2, 3)[2::myBool]`, `vm/index/b`},
		{`(1, 2, 3)[myNegative::2]`, `vm/slice/tuple/c`},
		{`(1, 2, 3)[three::2]`, `vm/slice/tuple/d`},
		{`(1, 2, 3)[0::bigNumber]`, `vm/slice/tuple/e`},
		{`ixE myBool, false`, `vm/user`},
		{`ixE false, myBool`, `vm/user`},
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
		{`foo [1, 2, 3], myBool`, `vm/index/i`},
		{`foo [1, 2, 3], myNegative`, `vm/index/j`},
		{`foo true, myNegative`, `vm/index/q`},
		{`ixs myColor, charm`, `vm/index/u`},
	}
	test_helper.RunTest(t, "index_test.pf", tests, test_helper.TestValues)
}
func TestInnerFunctionsAndVariables(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 42`, `42`},
		{`zort 3, 5`, `(25, 15)`},
		{`troz 2`, `2200`},
	}
	test_helper.RunTest(t, "inner_test.pf", tests, test_helper.TestValues)
}
func TestInterface(t *testing.T) {
	tests := []test_helper.TestItem{
		{`BLERP in Addable`, `true`},
		{`Fnug(5) in Addable`, `true`},
		{`ZORT in Foobarable`, `true`},
		{`true in Addable`, `false`},
		{`Fnug(5) in Foobarable`, `false`},
		{`Grunt(1, Derp(5))`, `Grunt(1, Derp(5))`},
		{`Derp(5) in Zort`, `true`},
		{`Derp(5) in Spoitable`, `true`},
		{`xuq Derp(5)`, `Derp(5)`},
		{`respoit Derp(5)`, `Derp(5)`},
	}
	test_helper.RunTest(t, "interface_test.pf", tests, test_helper.TestValues)
}
func TestImports(t *testing.T) {
	tests := []test_helper.TestItem{
		{`qux.square 5`, `25`},
		{`type qux.Color`, `type`},
		{`qux.RED`, `qux.RED`},
		{`type qux.RED`, `qux.Color`},
		{`qux.RED in qux.Color`, `true`},
		{`qux.Color(4)`, `qux.BLUE`},
		{`qux.Person "John", 22`, `qux.Person("John", 22)`},
		{`qux.Tone LIGHT, BLUE`, `qux.Tone(qux.LIGHT, qux.BLUE)`},
		{`troz.sumOfSquares 3, 4`, `25`},
	}
	test_helper.RunTest(t, "import_test.pf", tests, test_helper.TestValues)
}
func TestJson(t *testing.T) {
	if runtime.GOOS == "windows" {
		return
	}
	tests := []test_helper.TestItem{
		{`decode "25"`, `25`},
		{`decode "42.9"`, `42.9`},
		{`decode FOO`, `"foo"`},
		{`decode "false"`, `false`},
		{`decode "true"`, `true`},
		{`decode "null"`, `NULL`},
		{`decode "[1, 2, 3]"`, `[1, 2, 3]`},
		{`(decode MAP) == map("a"::1, "b"::2)`, `true`},
		{`decode "null"`, `NULL`},
		{`decode JOHN as Person`, `Person("John", 22)`},
		{`decode FRED as Person`, `Person("Fred", NULL)`},
		{`decode PEOPLE like list{Person}`, `[Person("John", 22), Person("Fred", NULL)]`},
		{`decode PEOPLE as list{Person}`, `list{Person}[Person("John", 22), Person("Fred", NULL)]`},
		{`decode PEOPLE_MAP as map{string, Person} == map{string, Person}("fred"::(Person("Fred", NULL)), "john"::(Person("John", 22)))`, `true`},
		{`decode PEOPLE_MAP like map{string, Person} == map("fred"::(Person("Fred", NULL)), "john"::(Person("John", 22)))`, `true`},
	}
	test_helper.RunTest(t, "json_test.pf", tests, test_helper.TestValues)
}

func TestLabels(t *testing.T) {
	tests := []test_helper.TestItem{
		{`label "qux"`, `qux`},
		{`label badString`, `vm/label/exists`},
	}
	test_helper.RunTest(t, "labels_test.pf", tests, test_helper.TestValues)
}
func TestLambdas(t *testing.T) {
	tests := []test_helper.TestItem{
		{`apply DOUBLE, 1`, `2`},
		{`apply double, 1`, `2`},
		{`apply foo, 1`, `vm/apply/func`},
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
		{`42.0`, `42.0`},
		{`42`, `42`},
		{`0b101010`, `42`},
		{`0o52`, `42`},
		{`0x2A`, `42`},
		{`NULL`, `NULL`},
		{`OK`, `OK`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
}

func TestLog(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/log.pf"`, `Starting script [36m"log.pf"[39m as service [36m"log"[39m.`},
		{`big 6`, `"small"`},
		{`hub log`, "\x1b[0m  ▪ Log at line 8 : Called \x1b[0m\x1b[48;2;0;0;64m\x1b[97mbig\x1b[0m. \n\x1b[0m  ▪ At line 9 we evaluated the condition \x1b[0m\x1b[48;2;0;0;64m\x1b[97mi >= 100\x1b[0m. The condition failed. \n\x1b[0m  ▪ At line 11 we took the \x1b[0m\x1b[48;2;0;0;64m\x1b[97melse\x1b[0m branch, so at line 12 function \x1b[0m\x1b[48;2;0;0;64m\x1b[97mbig\x1b[0m returned \x1b[0m\x1b[48;2;0;0;64m\x1b[97m\"small\"\x1b[0m."},
		{`hub halt "log"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestLogging(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 8`, test_helper.Foo8Result},
		{`foo 13`, test_helper.Foo13Result},
		{`qux 8`, test_helper.Qux8Result},
		{`qux 13`, test_helper.Qux13Result},
	}
	test_helper.RunTest(t, "logging_test.pf", tests, test_helper.TestOutput)
}

func TestLoggingToFile(t *testing.T) {
	currentDirectory, _ := os.Getwd()
	absoluteLocationOfLogFile, _ := filepath.Abs(currentDirectory + "/../compiler/test-files/logtest.md")
	os.Remove(absoluteLocationOfLogFile)
	tests := []test_helper.TestItem{
		{`qux 3`, `"odd"`},
	}
	test_helper.RunTest(t, "logging_to_file_test.pf", tests, test_helper.TestValues)
	resultBytes, err := os.ReadFile(absoluteLocationOfLogFile)
	if err != nil {
		t.Fatalf("unable to read file: %v", err)
	}
	if string(resultBytes) != test_helper.LogToFileResult {
		t.Fatal("Expected:\n", test_helper.LogToFileResult, "\nGot\n", string(resultBytes))
	}
}
func TestOverloading(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo 42`, `"int"`},
		{`foo "zort"`, `"string"`},
		{`foo 42, true`, `"any?, bool"`},
		{`foo 42.0, true`, `"any?, bool"`},
		{`foo true, true`, `"bool, bool"`},
	}
	test_helper.RunTest(t, "overloading_test.pf", tests, test_helper.TestValues)
}
func TestParameterizedTypes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Z{12}`, `Z{12}`},
		{`Z{5} == Z{12}`, `false`},
		{`Z{5}(3) + Z{5}(4)`, `Z{5}(2)`},
		{`Vec{3}[1, 2, 3] + Vec{3}[4, 5, 6]`, `Vec{3}[5, 7, 9]`},
		{`Money{USD} == Money{EURO}`, `false`},
		{`Money{USD}(3, 50)`, `Money{USD}(3, 50)`},
		{`Dragon{PURPLE}("Smaug", 500)`, `Dragon{PURPLE}("Smaug", 500)`},
		{`list{int}[1, 2]`, `list{int}[1, 2]`},
		{`list{int}[1, 2] + list{int}[3, 4]`, `list{int}[1, 2, 3, 4]`},
		{`Z{5}(4) in Z{5}`, `true`},
		{`Z{5}(4) in Z{12}`, `false`},
		{`clones{int}`, `clones{int}`},
		{`Zort{0}(0::0)`, `Zort{0}(0::0)`},
		{`Troz{0}(0)`, `Troz{0}(0)`},
		{`fooify one`, `vm/param/exist`},
	}
	test_helper.RunTest(t, "parameterized_type_test.pf", tests, test_helper.TestValues)
}

func TestPeek(t *testing.T) {
	tests := []test_helper.TestItem{
		{`peek c : 2 + 2`, `4`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestValues)
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
func TestRef(t *testing.T) {
	tests := []test_helper.TestItem{
		{`x ++`, `OK`},
	}
	test_helper.RunTest(t, "ref_test.pf", tests, test_helper.TestValues)
}
func TestReflection(t *testing.T) {
	tests := []test_helper.TestItem{
		{`reflect.isStruct Varchar{8}`, `false`},
		{`reflect.isClone Varchar{8}`, `true`},
		{`reflect.parent Varchar{8}`, `string`},
		{`reflect.parameterTypes Varchar{8}`, `[int]`},
		{`reflect.parameterValues Varchar{8}`, `[8]`},
	}
	test_helper.RunTest(t, "reflect_test.pf", tests, test_helper.TestValues)
}

func TestSnippet(t *testing.T) {
	tests := []test_helper.TestItem{
		{`(qux 5)[0]`, `"foo "`},
		{`(qux 5)[1]`, `10`},
		{`(qux 5)[2]`, `" bar"`},
		{`snippet(1, "q", true)`, `snippet(1, "q", true)`},
		{`len snippet(1, "q", true)`, `3`},
	}
	test_helper.RunTest(t, "snippets_test.pf", tests, test_helper.TestValues)
}

func TestSql(t *testing.T) {
	if runtime.GOOS == "windows" {
		return
	}
	tests := []test_helper.TestItem{
		{`testA`, `2`},
		{`testB`, `2`},
		{`testC`, `2`},
		{`testD`, `2`},
		{`testE`, `Dragon("Smaug", RED)`},
		{`testF`, `"Puff"::GREEN`},
		{`testG`, `map("Puff"::GREEN)`},
		{`testH`, `2`},
		{`testI`, `OtherData(NonEmptyString("foo"), NonZeroInt(42))`},
	}
	test_helper.RunTest(t, "sql_test.pf", tests, test_helper.TestOutput)
}

func TestSqlErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		return
	}
	tests := []test_helper.TestItem{
		{`mapKeyError`, `sql/concrete/map/key`},
		{`mapValueError`, `sql/concrete/map/value`},
		{`mapConflictError`, `sql/map/exists`},
		{`listError`, `sql/concrete/list`},
		{`setError`, `sql/concrete/set`},
		{`sigError`, `sql/sig`},
	}
	test_helper.RunTest(t, "sql_error_test.pf", tests, test_helper.TestValues)
}

func TestStructs(t *testing.T) {
	tests := []test_helper.TestItem{
		{`doug`, `Person("Douglas", 42)`},
		{`tom in Cat`, `true`},
		{`doug with age::43`, `Person("Douglas", 43)`},
		{`myCat[myField]`, `"Felix"`},
	}
	test_helper.RunTest(t, "struct_test.pf", tests, test_helper.TestValues)
}

func TestTests(t *testing.T) {
	tests := []test_helper.TestItem{
		{`test`, `OK`},
	}
	test_helper.RunTest(t, "test_test.pf", tests, test_helper.TestValues)
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
func TestTypeAccessErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Pair 1, 2`, `comp/private/call.a`},
		{`Suit`, `comp/private/type.a`},
		{`HEARTS`, `comp/private/ident`},
		{`one`, `comp/private/ident`},
	}
	test_helper.RunTest(t, "user_types_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestTypeInstances(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Z{3}(2) in Z{3}`, `true`},
		{`Z{5}(2) in Z{5}`, `true`},
		{`Z{7}(2) in Z{7}`, `true`},
		{`Z{12}(2) in Z{12}`, `true`},
	}
	test_helper.RunTest(t, "type_instances_test.pf", tests, test_helper.TestValues)
}
func TestUnwrapRtes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`unwrap foo x`, `vm/unwrap`},
	}
	test_helper.RunTest(t, "unwrap_test.pf", tests, test_helper.TestValues)
}
func TestUserDefinedTypes(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Tone with (shade::LIGHT, color::RED)`, `Tone(LIGHT, RED)`},
		{`Color(4)`, `BLUE`},
		{`DARK_BLUE`, `Tone(DARK, BLUE)`},
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
		{`Tone(LIGHT, GREEN)`, `Tone(LIGHT, GREEN)`},
		{`Tone(LIGHT, GREEN) == DARK_BLUE`, `false`},
		{`Tone(LIGHT, GREEN) != DARK_BLUE`, `true`},
		{`troz DARK_BLUE`, `Tone(DARK, BLUE)`},
		{`foo 3, 5`, `8`},
	}
	test_helper.RunTest(t, "user_types_test.pf", tests, test_helper.TestValues)
}
func TestValid(t *testing.T) {
	tests := []test_helper.TestItem{
		{`foo three`, `4`},
		{`foo zero`, `Error`},
	}
	test_helper.RunTest(t, "valid_test.pf", tests, test_helper.TestValues)
}
func TestValidation(t *testing.T) {
	tests := []test_helper.TestItem{
		{`EvenNumber x`, `EvenNumber(2)`},
		{`EvenNumber y`, `vm/validation/fail`},
		{`Person "Doug", goodNum`, `Person("Doug", 42)`},
		{`Person badString, 42`, `vm/validation/fail`},
		{`Person "Doug", neg`, `vm/validation/fail`},
		{`Thing zero`, `vm/user`},
		{`Thing one`, `vm/validation/bool`},
		{`Thing x`, `vm/validation/fail`},
		{`Thing y`, `Thing(3)`},
	}
	test_helper.RunTest(t, "validation_test.pf", tests, test_helper.TestValues)
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
func TestWith(t *testing.T) {
	tests := []test_helper.TestItem{
		{`john with name::"Susan", age::23`, `Person("Susan", 23)`},
		{`john with age::23`, `Person("John", 23)`},
		{`rex with [friends, 1]::"Daisy"`, `Dog("Rex", ["Fido", "Daisy"])`},
		{`Person with (name::"John")`, `Person("John", NULL)`},
		{`myList with diffList`, `["x", "y", "c", "d"]`},
		{`myOtherList with [2,1]::"q"`, `["a", "b", ["x", "q", "z"], "d"]`},
		{`myMap with "a"::99`, `map("a"::99, "b"::2, "c"::3, "d"::4)`},
		{`myMap with "z"::42`, `map("a"::1, "b"::2, "c"::3, "d"::4, "z"::42)`},
		{`myMap with diffMap`, `map("a"::99, "b"::99, "c"::3, "d"::4)`},
		{`otherMap with ["a", 1]::99`, `map("a"::[0, 99], "b"::[2, 3])`},
		{`myMap with "a"::99, "z"::42`, `map("a"::99, "b"::2, "c"::3, "d"::4, "z"::42)`},
		{`myMap without "a"`, `map("b"::2, "c"::3, "d"::4)`},
		{`myMap without "a", "b"`, `map("c"::3, "d"::4)`},
		{`badType with "foo"::99`, `vm/with/type/a`},
		{`intType with "foo"::99`, `vm/with/type/b`},
		{`Person with badString::99`, `vm/with/type/d`},
		{`Person with friends::badNum`, `vm/with/type/e`},
		{`myList with badVal::"foo"`, `vm/with/a`},
		{`myList with -1::"foo"`, `vm/with/b`},
		{`myList with 6::"foo"`, `vm/with/b`},
		{`myMap with F::"foo"`, `vm/with/c`},
		{`Cat with (badFieldsA)`, `vm/with/type/g`},
		{`Cat with (badFieldsB)`, `vm/with/type/h`},
		{`john with badList::23`, `vm/with/struct/b`},
		{`john with name::badNum`, `vm/with/f`},
		{`myOtherList with badList::"q"`, `vm/with/list/b`},
		{`myMap with badList::99`, `vm/with/map/b`},
	}
	test_helper.RunTest(t, "with_test.pf", tests, test_helper.TestValues)
}
func TestWrappers(t *testing.T) {
	if runtime.GOOS == "windows" {
		return
	}
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
