## README for the initializer

The `initializer` package supplies an `Initializer` struct which contains the data and supplies the methods necessary to initialize a Pipefish script, by creating and directing a parser and a compiler to operate on a VM. The `Initializer` returns a `Compiler` capable of dealing with runtime compilation of requests from the REPL: the `Initializer` can then be discarded together with its data.

There is one `Initializer` per module, as with compilers and parsers: an `Initializer` can spawn further `Intitializer`s recursively to initialize modules and external services.

The `initializer` package consists of the following files:

* `api_deserialization` is used to deserialize the APIs of external services.

* `api_serialization` is used to serialize the API of the service at compile time to supply to client services.

* `externals` contains everything else the initializer needs to set up ways for the VM to use external services.

* `function table` defines the `FunctionTable` type and methods for manipulating it. This is an intermediate step in the production of the `FunctionTree` objects that the compiler uses to perform multiple dispatch.

* `getters` supplies some miscellaneous utility functions for getting and transforming data.


* `gogen.go` which generates Golang source files.

* `gohandler.go` which does housekeeping for the Go interop.

* `initializer`, the main file directing initialization.

* `parsing` handles the early stages of intialization mostly concerned with setting up the parsers and parsing the code.

* `pchunks` defines things satifying the `parsedCode` interface. These are structured bundles of parsed code and metadata and so on representing function/command declarations, constant/variable declarations,
and type validation logic after the coe in them has been turned into ASTs

* `tchunks` defines things satifying the `tokenizedCode` interface. These are structured bundles of tokens and containers containing tokens, etc, representing the various kinds of declaration as identified by Pipefish's headwords, functions/commands, type declarations, import declarations, etc.

Fields of note in the `Initializer` struct are its compiler (naturally); its parser (a shortcut to the parser of the compiler); `Common`, a bindle of data that all the initializers of all the modules need to share; and `GoBucket`, which is used to accumulate the miscellaneous data swept up during parsing that we need to generate Go source files.