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

var StandardLibraries = dtypes.MakeFromSlice([]string{"crypto/aes", "crypto/bcrypt", 
	"crypto/rand", "crypto/rsa", "crypto/sha_256", "crypto/sha_512", "encoding/csv", 
	"encoding/base_32", "encoding/base_64", "encoding/json", "files", "fmt", "html", 
	"lists", "math", "math/big", "math/cmplx", "math/rand", "net/http", "net/mail", 
	"net/smtp", "net/url", "os/exec", "path", "path/filepath", "reflect", "regexp", 
	"sql", "strings", "terminal", "time", "unicode"})

const (
	OMIT_BUILTINS      = false // If true then the file builtins.pf, etc, will not be added to the service. Note that this means the hub won't work.
	IGNORE_BOILERPLATE = true  // Should usually be left true. Means that the flags below won't show instrumentation when compiling buitins.pf, etc.

	FUNCTION_TO_PEEK = "" // Shows the function table entry and function tree associated with the function named in the string, if non-empty.

	// These do what it sounds like.
	SHOW_LEXER             = false
	SHOW_RELEXER           = false
	SHOW_PARSER            = false // Note that this only applies to the REPL and not to code initialization. Use FUNCTION_TO_PEEK to look at the AST of a function.
	SHOW_INITIALIZER       = false
	SHOW_COMPILER          = false
	SHOW_COMPILER_COMMENTS = false // Note that SHOW_COMPILER must also be true for this to work.
	SHOW_RUNTIME           = false // Note that this will show the hub's runtime too at present 'cos it can't tell the difference. TODO.
	SHOW_RUNTIME_VALUES    = false // Shows the contents of memory locations on the rhs of anything (i.e. not the dest).
	SHOW_XCALLS            = false
	SHOW_GOLANG            = false
	SHOW_API_SERIALIZATION = false
	SHOW_EXTERNAL_STUBS    = false
	SHOW_TESTS             = true // Says whether the tests should say what is being tested, useful if one of them crashes and we don't know which.
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
