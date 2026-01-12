package test_helper_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

// The only way to test the test helperis to run one of each of the tests it supports. Which we
// were going to do anyway.

func TestAssignmentErrorsInSource(t *testing.T) {
	tests := []test_helper.TestItem{
		{``, `comp/assign/type/a`},
	}
	test_helper.RunTest(t, "assignment_error_test.pf", tests, test_helper.TestInitializationErrors)
}
func TestBooleanCompilerErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`5 or true`, `comp/bool/or/left`},
	}
	test_helper.RunTest(t, "compile_time_errors_test.pf", tests, test_helper.TestCompilerErrors)
}
func TestConstOrVarChunking(t *testing.T) {
	tests := []test_helper.TestItem{
		{"a int = 2 + 2", `a int = 3 tokens.`},
	}
	test_helper.RunInitializerTest(t, tests, test_helper.TestConstOrVarChunking)
}
func TestExternalOrImportChunking(t *testing.T) {
	tests := []test_helper.TestItem{
		{"foo::\"bar\"", `foo::"bar"`},
	}
	test_helper.RunInitializerTest(t, tests, test_helper.TestExternalOrImportChunking)
}
func TestFunctionChunking(t *testing.T) {
	tests := []test_helper.TestItem{
		{"qux : \n\t2 + 2\ngiven : 42\n", `qux : 5 tokens; given : 1 tokens.`},
	}
	test_helper.RunInitializerTest(t, tests, test_helper.TestFunctionChunking)
}
func TestHighlighter(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Type`, `[38;2;78;201;176mType[0m`},
	}
	test_helper.RunTest(t, "highlighter_test.pf", tests, test_helper.TestHighlighter)
}
func TestImperative(t *testing.T) {
	tests := []test_helper.TestItem{
		{`zort false`, `7`},
	}
	test_helper.RunTest(t, "imperative_test.pf", tests, test_helper.TestOutput)
}
func TestParserErrors(t *testing.T) {
	tests := []test_helper.TestItem{
		{`2 +`, `parse/prefix`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestParserErrors)
}
func TestParserOutput(t *testing.T) {
	tests := []test_helper.TestItem{
		{`x = 'q'`, `(x = 'q')`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestParserOutput)
}
func TestPrettyPrint(t *testing.T) {
	tests := []test_helper.TestItem{
		{`func(x) : x`, "func(x any?) :\n    x"},
	}
	test_helper.RunTest(t, "prettyprint_test.pf", tests, test_helper.TestPrettyPrinter)
}
func TestReparser(t *testing.T) {
	tests := []test_helper.TestItem{
		{`x`, `(x foo)`},
	}
	test_helper.RunTest(t, "", tests, test_helper.TestReparser)
}
func TestServices(t *testing.T) {
	test := []test_helper.TestItem{
		{"2 + 2", "4"},
	}
	test_helper.RunHubTest(t, "default", test)
}
func TestSigChunking(t *testing.T) {
	tests := []test_helper.TestItem{
		{`qux (a, b) foo :`, `qux (a any?, b any?) foo`},
	}
	test_helper.RunInitializerTest(t, tests, test_helper.TestSigChunking)
}
func TestTeardown(t *testing.T) {
	// no t.Parallel()
	test_helper.Teardown("teardown_test.pf")
	tests := []test_helper.TestItem{
		{`2 + 2`, `4`},
	}
	test_helper.RunTest(t, "teardown_test.pf", tests, test_helper.TestValues)
	test_helper.Teardown("teardown_test.pf")
}
func TestTypeChunking(t *testing.T) {
	tests := []test_helper.TestItem{
		{"Number = abstract int/float", `Number = abstract int/float`},
	}
	test_helper.RunInitializerTest(t, tests, test_helper.TestTypeChunking)
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
	test_helper.RunTest(t, "", tests, test_helper.TestTypeParserOutput)
}
func TestValues(t *testing.T) {
	tests := []test_helper.TestItem{
		{`A`, `42`},
	}
	test_helper.RunTest(t, "variables_test.pf", tests, test_helper.TestValues)
}
