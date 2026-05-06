# Pipefish for BenchGen.

## Note about headwords

In Pipefish, the keywords saying what sort of thing you're declaring (`newtype`, `cmd`, `import`, etc) have scope until the next such word. This is so that lazy people like me can e.g. define functions by writing:

```
def

foo(x int) :
    <body of foo>

bar(y bool) :
    <body of bar>

qux(x string) :
    <body of qux>
```

However, you can also write a headword for every function (or new type or import or whatever you're declaring) and in the same line, like this, and this will I think be much more convenient for generated code:

```
def foo(x int) :
    <body of function>

def bar(y bool) :
    <body of function>

def qux(x string) :
    <body of function>
```

[Wiki: headwords](https://github.com/tim-hardcastle/Pipefish/wiki/Comments,-continuations,-and-headwords)

## Functions and conditionals

Let's start with what functions look like and the conditional syntax, since these things go together. Functions are declared under the `def` headword like:


```
foo(x int, y string, z bool) :
    <body of function>
```

If you leave the types off, the parameters are of type `any?`. (It runs faster if you put the types on, it can figure more stuff out at compile-time.) A return signature is optional:

```
foo(x int, y string, z bool) -> string :
    <body of function>
```

if-then-else, if written in a single line, would look like `foo or bar : zort ; else : troz`. But a newline can be used for the `;`.

```
classifyNumber(i int) -> string :
    i > 0 :
        i < 10 :
            "small positive number"
        else :
            "large positive number"
    i < 0 :
        i > -10 :
            "small negative number"
        else :
            "large negative number"
    else :
        "zero"
```

Since the body of a Pipefish function is just a single expression, they all basically look like that. It's a very simple language.

However, to make it work, we also need local variables. These are declared in the (optional) `given` block of a function, along with any local functions you want.

```
gcd(a, b int) :
    remainder == 0 :
        b
    else :
        gcd b, remainder
given :
    remainder = a mod b
```

This separates two things that should never ever have gotten mixed up in the first place --- (a) giving names to our concepts (b) flow of control.

Note that locals are evaluated by need.

Functions can be overloaded and have multiple dispatch.

[Wiki: introducing functions](https://github.com/tim-hardcastle/Pipefish/wiki/Introducing-functions)

## For loops

Pipefish has pure `for` loops with immutable, referentially-transparent variables. (There is [prior art](https://futhark-lang.org/blog/2026-01-20-why-not-tail-recursion.html) in the functional parallelization language Futhark.)

This takes a minute or two to get used to, after which hopefully you will fall in love as I did. Here's an example of a function finding the nth triangular number.

```
triang(n int) :
    from a = 0 for _::i = range 1::n+1 : 
        a + i
```

We'll call `a` the *bound variable* and `i` the *index variable* (mathematically they are both bound variables, but it's convenient to distinguish between them). You can see what's happening to `i` each time we go round the loop. What's happening to `a` (from a procedural point of view) is that it's being replaced by the expression in the body of the loop. Every time we go round, we let `a` equal `a + i`.

(We write `_::i` because all `range` style loops range over a key and a value, and we're discarding the key with `_`, as in Golang.)

We can also write `for` loops C-style:

```
triang(n int) :
    from a = 0 for i = 1; i <= n; i + 1 : 
        a + i
```

Just as `a + i` says what happens to the bound variable each time we go round the loop, so here `i + 1` says what happens to the index variable.

We can have more than one bound variable:

```
fib(n int) :
    from a, b = 0, 1 for i = 0; i < n; i + 1 :
        b, a + b
```

However, if you try this out you will find that this returns *two* numbers. Idiomatically, when we want to return a single value from a `for` loop with multiple returns, we use the built-in function `first`, which selects the first member of a tuple.

```
fib(n int) :
    first from a, b = 0, 1 for i = 0; i < n; i + 1 :
        b, a + b
```

We also have `break` and `continue`. `break` means that the `for` loop stops and returns whatever the index variables are; `break <expression>` means that the `for` loop stops and returns whatever `<expression>` evaluates to; and `continue` means that the bound variables are unchanged. E.g. 

```
find(L list, x any?) -> int? :
    from a int? = NULL for i::el = range L :
        type(el) == type(x) and el == x :
            break i
        else :
            continue 
```

`for` loops can also have `given` blocks like functions. An example from [a recent project](https://github.com/tim-hardcastle/pipefish/tree/main/examples/rofl), which finds the fixed-point of applying a list of rules to an expression.

```
evaluate(rules list, exp string) :
    ruleOrValue from a = exp for :
        a == newExp :
            break
        else :
            newExp
    given :
        newExp = apply(rules, a)
```

This means that unlike most other functional languages we can use a `for` loop whenever we would in an imperative language, but with fewer footguns. We only need to use recursion when we have data-structures like trees, and higher-order-functions if we're doing something extremely clever.

[Wiki: for loops](https://github.com/tim-hardcastle/pipefish/wiki/For-loops)

## Data types

We have the basic stuff you'd expect, `string`, `bool`, `int`, `float`, `rune` (a Unicode codepoint like in Go).

You ask specifically about arrays: there are no fixed-length arrays. There are just things of type `list`. The literals are with `[...]` as in Python: `[1, 2, 3]`, `["foo", 42, true]`, `[]`. There can be lists of lists.

There are also no pointers: it's a functional language, the semantics are by-value.

Struct types are declared under the `newtype` headword like `Person = struct(name string, age int)`. Then `Person` is also a constructor function `joe = Person("Joseph", 22)`.

Structs are indexed with square brackets (`joe[name]`), like everything else, because the labels of structs are first-class values.

All values are immutable, so you can't assign to the fields. Instead you use the `with` construction to clone-and-modify the value: `joe with age::23`. (The same syntax copy-and-modfies lists, `myList with 3::"foo"`.)

[Wiki: container types](https://github.com/tim-hardcastle/Pipefish/wiki/Container-types)
[Wiki: structs](https://github.com/tim-hardcastle/Pipefish/wiki/Structs)
[Wiki: with and without](https://github.com/tim-hardcastle/Pipefish/wiki/With-and-without)

## The RNG

Pipefish enforces [functional-core/imperative-shell](https://github.com/tim-hardcastle/pipefish/blob/main/docs/functional-core-imperative-shell.md) semantics, and because getting random number is impure and so imperative, it belongs in the imperative shell.

There is a math/rand library, with the API documented [here](https://github.com/tim-hardcastle/Pipefish/wiki/The-rand-library). Note that conventionally when you import a library that's impurity-heavy you import it without a namespace, because it's always obvious what you're doing.

```
import

NULL::"math/rand"
```

And then we can write, not a function, but a *command*, declared under the `cmd` headword, to get random numbers from the RNG.

```
postTenRandomNumbers :
    for i = 0; i < 10; i + 1 :
        get number from Random 1::11 
        post number
```

(Usually you'd want to do something with your random numbers by passing them to your business logic in the functions, but here we just post them to output.)

[Wiki: imports](https://github.com/tim-hardcastle/Pipefish/wiki/Imports-and-libraries)
[Wiki: the rand library](https://github.com/tim-hardcastle/Pipefish/wiki/The-rand-library)
[Wiki: imperative Pipefish](github.com/tim-hardcastle/Pipefish/wiki/Imperative-Pipefish)
      
---

You now know quite a lot of Pipefish. If you look at e.g. the little [text-based adventure game](https://github.com/tim-hardcastle/pipefish/blob/main/examples/adventure/adv.pf) in the examples folder, you'll find it readable: it's just a bunch of functions, conditionals, and `for` loops, just like all the other Pipefish.
