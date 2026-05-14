// All this does is contain in one place the constants controlling which bits of the inner workings of the
// lexer/parser/compiler/are displayed to me for debugging purposes. In a release they must all be set to false
// except SUPPRESS_BUILTINS which may as well be left as true.

package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tim-hardcastle/pipefish/source/dtypes"
)

// This can be changed during initialization.
var MandatoryImports = []string{"rsc-pf/builtins.pf", "rsc-pf/interfaces.pf", "rsc-pf/generics.pf"}

// And so this is a function. TODO --- init it instead.
func MandatoryImportSet() dtypes.Set[string] {
	return dtypes.MakeFromSlice(MandatoryImports)
}

var ThingsToIgnore = (dtypes.MakeFromSlice(MandatoryImports)).
	Add("user/hub/hub.hub").Add("user/hub/hub.pf").Add("source/hub/hub.pf").
	Add("Builtin constant").Add("user/themes.pf")

const (
	OMIT_BUILTINS      = false // If true then the file builtins.pf, etc, will not be added to the service. Note that this means the hub won't work.
	IGNORE_BOILERPLATE = true  // Should usually be left true. Means that the flags below won't show instrumentation when compiling buitins.pf, etc.

	FUNCTION_TO_PEEK = "" // Shows the function table entry and function tree associated with the function named in the string, if non-empty.

	// We want all the peeking of the VM to depend on whether this constant is true, so that setting it to false will mean that this
	// logic doesn't even compile, and won't slow down the VM.
	PEEK_VM = true
	// Ditto for the compiler, we don't care how fast it is yet but we will one day.
	PEEK_COMPILER = true
	// Path relative to the root of the repo to dump output to.
	DUMP_PATH = "dump.txt"

	// These do what it sounds like.
	SHOW_LEXER             = false
	SHOW_RELEXER           = false
	SHOW_PARSER            = false // Note that this only applies to the REPL and not to code initialization. Use FUNCTION_TO_PEEK to look at the AST of a function.
	SHOW_INITIALIZER       = false
	SHOW_XCALLS            = false
	SHOW_GOLANG            = false
	SHOW_API_SERIALIZATION = false
	SHOW_EXTERNAL_STUBS    = false
	SHOW_TESTS             = false // Says whether the tests should say what is being tested, useful if one of them crashes and we don't know which.
	SHOW_BLING_TREE        = false
	ALLOW_PANICS           = true // If turned on, permits panics in the vm instead of turning them into error messages.
	SHOW_ERRORS            = false
)

var PipefishHomeDirectory string

func init() {
	if testing.Testing() {
		currentDirectory, _ := os.Getwd()
		absolutePath, _ := filepath.Abs(currentDirectory + "/../../")
		PipefishHomeDirectory = absolutePath + "/"
	} else {
		appDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		PipefishHomeDirectory = appDir + "/"
	}
}
