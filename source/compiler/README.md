# README for the "compiler" package.

The compiler package contains everything we need to compile a Pipefish *expression* to bytecode.

It doesn't know how to compile a Pipefish *script*, since this involves other tasks such as declaring types and constants and imports and top-level functions. This is the job of the initializer, which calls upon the compiler to compile expressions, such as the bodies of functions, `given` blocks, validation logic, etc.

## The test-files directory.

For reasons, possibly bad ones, the `.pf` test files initialized by the initializer, compiler, and VM for testing purposes are kept in here with the compiler and not in the `test_helper` module where you might expect them.

Besides this, the compiler has the following files.

## Files

* `builtin` contains the code for generating builtin functions.

* `compiler` knows everything needed to compile a line of code from the REPL at runtime. It does *not* know how to compile a script by itself, and is directed in this by the `initializer` package.

* `environment` supplies data structures and their getters and setters for keeping track of where in (virtual) memory variables are stored.

* `function_call` breaks out the logic for compiling a function call, which is complicated because of the multiple dispatch.

* `function_tree` defines the `FunctionTree` type which the compiler uses to perform multiple dispatch.

* `getters` is a miscellaneous collection of helper functions for extracting data conveniently from wherever its stored and converting it from one form to another. This can be quite convoluted as a result of my attempts to maintain a single source of truth.

* `tracking` contains types and methods for tracking and logging.

* `types` contains a rather miscellaneous collection of things for manipulating types, particularly abstract types.

* `typeschemes` contains things satisfying the `TypeScheme` interface, and mmethods and functions for manipulating them. These represent the compiler's view of the type system. This is somewhat richer than that enjoyed by the user or the compiled code, in that it can keep track of the types of the elements of tuples; and, eventually of pairs.

