package test_helper

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tim-hardcastle/pipefish/source/compiler"
	"github.com/tim-hardcastle/pipefish/source/err"
	"github.com/tim-hardcastle/pipefish/source/hub"
	"github.com/tim-hardcastle/pipefish/source/initializer"
	"github.com/tim-hardcastle/pipefish/source/parser"
	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/text"
	"github.com/tim-hardcastle/pipefish/source/values"
	"github.com/tim-hardcastle/pipefish/source/vm"
)

// Auxiliary types and functions for testing the parser and compiler.

type TestItem struct {
	Input string
	Want  string
}

func RunTest(t *testing.T, filename string, tests []TestItem, F func(cp *compiler.Compiler, s string) (string, error)) {
	wd, _ := os.Getwd() // The working directory is the directory containing the package being tested.
	for _, test := range tests {
		if settings.SHOW_TESTS {
			println(text.BULLET + "Running test " + text.Emph(test.Input))
		}
		var cp *compiler.Compiler
		if filename == "" {
			cp, _ = initializer.StartCompilerFromFilepath(filename, map[string]*compiler.Compiler{}, &values.Map{})
		} else {
			cp, _ = initializer.StartCompilerFromFilepath(filepath.Join(wd, "../compiler/test-files/", filename), map[string]*compiler.Compiler{}, &values.Map{})
		}
		got, e := F(cp, test.Input)
		if e != nil {
			println(text.Red(test.Input))
			r := cp.P.ReturnErrors()
			println("There were errors parsing the line: \n" + r + "\n")
		}
		if !(test.Want == got) {
			// if len(test.Want) != len(got) {
			// 	for i, ch := range test.Want {
			// 		println(ch, string(ch), got[i], string(got[i]))
			// 	}
			// }
			t.Fatalf("Test failed with input %s \nExp :\n%s\nGot :\n%s", test.Input, test.Want, got)
		}
	}
}

// NOTE: this is here to test some internal workings of the initializer. It only initializes
// a blank service.
func RunInitializerTest(t *testing.T, tests []TestItem, F func(iz *initializer.Initializer, s string) string) {
	iz := initializer.NewInitializer(initializer.NewCommonInitializerBindle(&values.Map{}, map[string]*compiler.Compiler{}))
	iz.ParseEverythingFromSourcecode(vm.BlankVm(), parser.NewCommonParserBindle(), compiler.NewCommonCompilerBindle(), "", "", "")
	for _, test := range tests {
		if settings.SHOW_TESTS {
			println(text.BULLET + "Running test " + text.Emph(test.Input))
		}
		got := F(iz, test.Input)
		if !(test.Want == got) {
			t.Fatalf("Test failed with input %s \nExp :\n%s\nGot :\n%s", test.Input, test.Want, got)
		}
	}
}

// These functions say in what to extract information from a compiler, given
// a line to put in: do we want to look at the returned value; or what was posted
// to output; or the errors in the compiler.

func TestValues(cp *compiler.Compiler, s string) (string, error) {
	if cp.P.Common.IsBroken {
		return cp.P.Common.Errors[0].ErrorId, errors.New(cp.P.Common.Errors[0].Message)		
	}
	v := cp.Do(s)
	if cp.ErrorsExist() {
		return "", errors.New("failed to compile with code " + cp.P.Common.Errors[0].ErrorId)
	}
	if v.T == values.ERROR {
		return v.V.(*err.Error).ErrorId, nil
	}
	return cp.Vm.Literal(v), nil
}

func TestHighlighter(cp *compiler.Compiler, s string) (string, error) {
	v := cp.Do(`DARK_MODERN`)
	return cp.Highlight([]rune(s), v.V.(*values.Map)), nil
}

func TestOutput(cp *compiler.Compiler, s string) (string, error) {
	if cp.P.Common.IsBroken {
		return cp.P.Common.Errors[0].ErrorId, errors.New(cp.P.Common.Errors[0].Message)		
	}
	cp.Vm.OutHandle = vm.MakeCapturingOutHandler(cp.Vm)
	ok := cp.Do(s)
	if ok.T == values.ERROR {
		return "", errors.New("runtime error with code " + ok.V.(*err.Error).ErrorId)
	}
	if cp.ErrorsExist() {
		return "", errors.New("failed to compile with code " + cp.P.Common.Errors[0].ErrorId)
	}
	return text.StripColors(cp.Vm.OutHandle.(*vm.CapturingOutHandler).Dump()), nil
}

// Tests for the error in a line of code, given successful compilation of the `_test.pf` file.`
func TestCompilerErrors(cp *compiler.Compiler, s string) (string, error) {
	if cp.P.Common.IsBroken {
		return cp.P.Common.Errors[0].ErrorId, errors.New(cp.P.Common.Errors[0].Message)		
	}
	if s == "" {
		return "comp/", nil
	}
	v := cp.Do(s)
	if !cp.ErrorsExist() {
		return "", errors.New("unexpected successful evaluation returned " + text.Emph(cp.Vm.Literal(v)))
	} else {
		return cp.P.Common.Errors[0].ErrorId, nil
	}
}

func TestInitializationErrors(cp *compiler.Compiler, s string) (string, error) {
	return cp.P.Common.Errors[0].ErrorId, nil
}

// These functions test the internal workings of the initializer.
func TestSigChunking(iz *initializer.Initializer, s string) string {
	iz.P.PrimeWithString("test", s)
	sig, ok := iz.ChunkFunctionSignature()
	if !ok {
		return "Couldn't parse sig."
	}
	return sig.SigAsString()
}

func TestFunctionChunking(iz *initializer.Initializer, s string) string {
	iz.P.PrimeWithString("test", s)
	fn, ok := iz.ChunkFunction(false, false, "")
	if !ok {
		return "Couldn't parse function."
	}
	return initializer.SummaryString(fn)
}

func TestTypeChunking(iz *initializer.Initializer, s string) string {
	iz.P.PrimeWithString("test", s)
	ty, ok := iz.ChunkTypeDeclaration(false, "")
	if !ok {
		return "Couldn't parse type."
	}
	return initializer.SummaryString(ty)
}

func TestConstOrVarChunking(iz *initializer.Initializer, s string) string {
	iz.P.PrimeWithString("test", s)
	ty, ok := iz.ChunkConstOrVarDeclaration(false, false, "")
	if !ok {
		return "Couldn't parse assignment."
	}
	return initializer.SummaryString(ty)
}

func TestExternalOrImportChunking(iz *initializer.Initializer, s string) string {
	iz.P.PrimeWithString("test", s)
	ty, ok := iz.ChunkImportOrExternalDeclaration(false, false, "")
	if !ok {
		return "Couldn't parse import/external declaration."
	}
	return initializer.SummaryString(ty)
}

var Foo8Result = "We called function `foo` - defined at line 13 - with `i` = `8`.\n" +
	"At line 14 we evaluated the condition `i mod 2 == 0`. \n" +
	"The condition succeeded.\n" +
	"At line 15 function `foo` returned \"even\".\n"

var Foo13Result = "We called function `foo` - defined at line 13 - with `i` = `13`.\n" +
	"At line 14 we evaluated the condition `i mod 2 == 0`. \n" +
	"The condition failed.\n" +
	"At line 16 we took the `else` branch.\n" +
	"At line 17 function `foo` returned \"odd\".\n"

var Qux8Result = "Log at line 7 : We're here.\n" +
	"Log at line 8 : We test to see if i (8) is even, which is true.\n" +
	"Log at line 9 : We return \"even\", because 8 is even.\n"

var Qux13Result = "Log at line 7 : We're here.\n" +
	"Log at line 8 : We test to see if i (13) is even, which is false.\n" +
	"Log at line 10 : Guess we're taking the 'else' branch.\n" +
	"Log at line 11 : And we return \"odd\".\n"

func Teardown(nameOfTestFile string) {
	currentDirectory, _ := os.Getwd()
	absolutePathToGobucket, _ := filepath.Abs(currentDirectory + "/../../source/initializer/gobucket/")
	locationOfGoTimes := absolutePathToGobucket + "/gotimes.dat"
	absoluteLocationOfPipefishTestFile, _ := filepath.Abs(currentDirectory + "/../compiler/test-files/" + nameOfTestFile)
	temp, err := os.ReadFile(locationOfGoTimes)
	if err != nil {
		panic("Couldn't read gotimes; error was " + err.Error())
	}
	timeList := strings.Split(strings.TrimRight(string(temp), "\n"), "\n")
	newTimes := ""
	for i := 0; i + 1 < len(timeList); i = i + 2 {
		if timeList[i] != absoluteLocationOfPipefishTestFile {
			newTimes = newTimes + timeList[i] + "\n" + timeList[i+1] + "\n"
		}
	}
	file, _ := os.Stat(absoluteLocationOfPipefishTestFile)
	timestamp := file.ModTime().UnixMilli()
	goTestFile := absolutePathToGobucket + "/" + text.Flatten(absoluteLocationOfPipefishTestFile) + "_" + strconv.Itoa(int(timestamp)) + ".so"
	os.Remove(goTestFile)
	os.WriteFile(locationOfGoTimes, []byte(newTimes), 0644)
}

type TestPair struct {
	Input  string
	Expect string
}

type capturingWriter struct {capture string} 

func (c *capturingWriter) get() string {
	s := c.capture 
	c.capture = ""
	return s
}

func (c *capturingWriter) Write(b []byte) (n int, err error) {
	c.capture = c.capture + string(b)
	return len(b), nil
}

func RunServiceTest(t *testing.T, hubName string, test []TestPair) { 
	wd, _ := os.Getwd() // The working directory is the directory containing the package being tested.
	sourceDir, _ := filepath.Abs(filepath.Join(wd, "/../")) // We may be calling this either from in the `hub` direcotry or `pf`.
	hubDir := filepath.Join(sourceDir, "hub/test-files", hubName)
	h := hub.New(hubDir, &capturingWriter{})
	for _, item := range test {
		h.Do(item.Input, "", "", "", false)
		result := strings.TrimSpace(h.Out.(*capturingWriter).get())
		if result != item.Expect {
			t.Fatal("\nOn input '" + item.Input + "'\n    Exp : '" + item.Expect + "'\n    Got : '" + result + "'")
		}
	}
}
