## ROFL: regex-oriented functional language

This is a very concise (but very silly) way of defining what is at least superficially a working language, in that 85 sloc of Pipefish code allows us to define a language where the following code does in fact define the given functions. (Note that the imported `math` library defines the `nat` type in binary, and so `10` means `2` in these examples.)

```
import lib/math
import lib/variables

double (@n: nat) ->
    10 * @n 

size (@n: nat) ->  
    if @n < 1100100 then
        small
    else
        large 

(@k: nat) mod (@n: nat) ->
    if @k < @n then
        @k 
    else 
        <@k - @n> mod @n 

fac (@n: nat) -> 
    if @n == 0 then 
        1 
    else 
        @n * fac <@n - 1>
```

This is achieved by regexes, low cunning, and some really terrible ideas.

## ROFL language specification

A ROFL script consists of `import` statements and expressions.

An *import statement* is of the form `import <filename>` and behaves as though the imported file had been pasted into the importing file at that point.

An *expression* is any other newline-terminated string. (As sugar, strings may be carried on over several lines by beginning the second and subsequent strings with whitespace.)

An **arrow** is the substring ` -> `. An expression with at least one arrow is called a *rule*, otherwise it is a *value*.

For a rule with *n* arrows, its *main arrow* is the ⌈n/2⌉ᵗʰ. This divides a rule into a left-hand and right-hand side: `lhs -> rhs`.

A rule can be applied to an expression by treating the lhs as a regex, the rhs as the replacement string, and using this to perform a "replace all" on the expression.

We can use a list of rules to *evaluate* an expression by going through the list of rules in order, applying each of them to the expression, and then doing that over and over until we reach a fixed expression.

A ROFL script is executed as follows.

* Begin with an empty list of rules.
* Go through the expressions/import statements in the script in order
* If we have an import statement, import the file if the file hasn't been imported, otherwise ignore it.
* Otherwise, it's an expression and we evaluate it in the context of the current list of rules.
* If the result of evaluating the expression is a value, post it to output (ignoring the empty string).
* Otherwise, it is a rule, and we add it to the list of rules.

That's it.