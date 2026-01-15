# Tokenization rules

Here we give the exact rules for how to form a Pipefish identifier — the sequence of characters which names a variable, a function, a type, a struct's field, etc.

We can divide Unicode characters into the following groups:

* Whitespace. You can't use this in an identifier.

* The "protected punctuation", the characters `(`, `)`, `[`, `]`, `{`, `}`, `,`, `;`, `:`, `.`, `"`, `` ` ``, `'` and `|`. You can't use any of these in an identifier.

* Alphabetic characters. You can use any of these in an identifier.

* The numerals `0` ... `9`. You can use these in an identifier, but you can't use them to start an identifier.

* Symbols, consisting of everything else except the underscore character _. You can use these in an identifier.

* The underscore, `_`. You can use this in an identifier, but you can't use it at the start or the end of an identifier (except in the special case of `_` being the whole of the identifier).

The rule governing their arrangement is that alphabetic characters can only go next to other alphabetic characters or the underscore; numerals can only go next to other numerals or the underscore; and symbols can only go next to other symbols or the underscore.

So the underscore acts as a bridge between the three other kinds of characters you can put in an identifier: `foo_+` is a legal identifier; `foo+` is not; `&_3` is a legal identifier but `&3` is not.

The upshot of this is that if people want to write `a/b` or `x+1` they can do so and this is unambiguous, since the letters and numbers must belong to different identifiers than the symbols. *However*, it may sometimes be useful to qualify a symbol by a letter or word or vice-versa. At this point one can write for example `~_R`, using the `_` as a neutral bridge between symbol and letter.

## Note

A little experimentation may make you think that you can break these rules. For example if you define :

```
def

(x int) ~R (y int) :
    x == -y 
```

then this will compile and work *almost* exactly how you think it will. However, you have *not* in this case defined an infix operator `~R`: rather, the definition above is equivalent to:

```
def

(x int) ~ R (y int) :
    x == -y 
```
