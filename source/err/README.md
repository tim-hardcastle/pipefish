This module generates errors for everything else from the lexer to the vm.

The main function is `CreateErr(errorId string, tok *token.Token, args ...any) *Error`, in `errors.go`.

This takes the `errorId` and uses it as a key to find functions in a map in `errorfile.go` which it then feeds the remaining arguments to generate the error message and a help message (i.e. what you get if you type `hub why <number>`).

Because this module is a dependency of the parser, vm, etc, it can't depend on any of them, and so e.g. it can't call the parser to prettyprint an expression, or the VM to describe a value. So these things are turned into strings which are passed in as the arguments.