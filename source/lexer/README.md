The lexer is fairly standard, but a little more complicated than an ordinary lexer to deal with whitespace and other quirks of the language. In particular, the lexer can return any number of tokens at a time.

The tokens are then passed through a bucket chain of "relexers" which tweak the raw tokenized output into something easier for the parser to parse.

FInally the stream of tokens is fed to a "monotokenizer" which emits tokens one by one when the parser requests them.
