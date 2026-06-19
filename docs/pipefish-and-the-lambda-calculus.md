## Pipefish and the lambda calculus

These are some very informal notes explaining the relationship of Pipefish to the lambda calculus.

The functions of Pipefish are written in heavily sugared lambda calculus, because [Pipefish is an ISWIM dialect](https://www.cs.cmu.edu/~crary/819-f09/Landin66.pdf). This is what the `given` blocks do for us, they're Pipefish's concrete syntax for Landin's **where**.

Pipefish has unusual conditionals for a functional language (or indeed any language). The expression `x == 42 : "foo"`, meaning "return `"foo"` if `x` is `42`", is perfectly valid. The operator `:` is lazily evaluated. If the LHS is true, it evaluates the RHS and returns what it evaluates to; otherwise it returns `UNSAT`, the sole member of the `unsat` type, meaning "unsatisfied conditional".

The operator `;` is also lazily evaluated; after evaluating the LHS we return what it evaluates to if it isn't `UNSAT`, otherwise we evaluate the RHS and return that value.

As sugar, we use a newline for `;`, and `else` as sugar for `true` on the LHS of `:`.

A `for` loop is a heavily-sugared way of saying "get the result of passing this closure (i.e. the body of the `for` loop) along with these other parameters (e.g. the range of the `for` loop, the initial values of the bound variables, etc) to this really powerful and polymorphic higher-order function", and could be explicitly desugared into that, and then desugared still further into the lambda calculus.

To deal with errors, we say that implicitly every parameter of every function/operator can be of type `error` besides the declared types, and that a function passed one or more errors as arguments will return the leftmost error immediately. Note that this rule also applies to the `;` operator.

We then supply error handling with a built-in function `valid` which is exempt from this rule and returns a boolean which is true if the type is not an error; and a built-in function `unwrap` which can convert an `error` into an `Error`, an ordinary struct type which contains the error message etc. (Or, to put it another way, we throw exceptions and catch them.)

A command is an expression with side-causes/effects, and *must* evaluate to `UNSAT`, to `OK` (the sole member of the `ok` type) or to an `error`.

`;` operates on `OK` differently from other values: `C1 ; C2` returns the value of `C1` if it evaluates to an error (of course), otherwise it evaluates to whatever `C2` evaluates to.

The side-causes and side-effects correspond to commands having the *state* as a side-condition; when a command is evaluated, whether it evaluates to `OK` or `UNSAT` or to an error (and if the latter, which error) may depend on the state; and the side-effects of the command may change the state.

We place a few sanity restrictions on the sort of functions and expressions that the user can write:

* To preserve the functional-core/imperative-shell semantics, the `;` operator returns an error if one of its operands is `OK` and the other is anything other than `OK`, `UNSAT` or `error`. (In practice, this is caught at compile-time.) As there's no way for users to make their own operators that take commands as operands other than the built-in `;`, there are no escape hatches.

* There's no way to define anything but `valid` and `unwrap` that can handle errors.

* We ensure that the user can't do anything with the `unsat` type, so that it can only be constructed by the lhs of a conditional being `false`, and can only be consumed by the `;` operator.