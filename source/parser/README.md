This is essentially a Pratt parser in which Everything Is An Expression, but with some additional bits and pieces.

* `ast` defines the nodes of the AST, satisfying the `Node` interface.

* `bling tree` keeps track of what bling is possible, i.e. if you declare a function `receipts for (c Customer)` then the parser can and will interpret `for` after `receipts` as bling and not as the start of a `for` loop.

* `getters` is a miscellaneous collection of helper functions for extracting data conveniently from wherever its stored and converting it from one form to another. 

* `parser_test` contains the tests for the package.

* `parse_types` deals with parsing type representations, e.g. `list{float/int}`, supplying them with their own specialized AST and parser.

* `precedence` contains constants and functions for dealing with the rules of precedence for parsing.

* `prettyprint` prettyprints an AST.

* `signature` defines a representation of a signature as names of variables paired with abstract types.

* `tchunks` contains a definition of the `TokenisedCodeChunk` type and all the things the parser needs to know about it: there are more functions in a corresponding `tchunks` file in the `initializer` package for initialization time.

* `type_ast` file supplies a seperate AST for representing type expressions such as `list{string/int}` which have their own rules.

