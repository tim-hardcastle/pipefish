This is essentially a Pratt parser in which Everything Is An Expression, but with some additional bits and pieces.

Most notably, there is the bling tree. This keeps track of what bling is possible, i.e. if you declare a function `receipts for (c Customer)` then the parser can and will interpret `for` after `receipts` as bling and not as the start of a `for` loop.

The `type_ast.go` file deals with parsing type sexpressions such as `list{string/int}` which have their own rules.

