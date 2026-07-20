Let's write a Pratt parser! (and a lexer, three interpreters, and two compilers)

## Introduction

This document is suitable as an introduction to some basic concepts for langdev beginners, but it may also give more experienced people things to think about. For one thing, a Pratt parser in a purely functional style is a thing of almost inconceivable simplicity and beauty; and also you may not have seen some of the tricks I can do with them.

## Terminology

Let's define some terms.

* A **lexer** (and/or **tokenizer**) takes code considered just as a string of characters, and converts it into the next higher unit of structure, the `token`, for example turning the string `"(-42 + 99) * 4!"` into the list `["(", "-", "42", "+", "99", ")", "*", "4", "!"]`.

* Normally instead of the tokens being a list of mere bare strings they have **metadata** to say where in the code they came from: in a mature language, the name of the source code file, the line number, and the position in the line.

* A **parser** takes this string of tokens and turns it into a data structure that more accurately represents the *syntactic* structure of the code. The most usual form, and the one we will use here, is to turn the expression into an **abstract syntax tree**, or **AST**, which assigns the tokens to nodes of a tree such that the branches of a node are the arguments of an operation. For example, from the tokens `["(", "-", "42", "+", "99", ")", "*", "4", "!"]`, the AST would be:
```
                 *
                / \
               /   \
              +     !
             / \    |
            |   99  4
            -
            |
            42                
```
 Note that the parentheses have disappeared, because the structure of the AST itself represents the correct grouping of operations.

 A **compiler**, starting from this high point of abstraction, turns the AST into something that is more convenient for the computer to actually execute. This may be the machine code of the computer, or it may be **bytecode**, that is, code that mimics the structure of machine code in being a flat list of instructions to be executed sequentially.

 By contrast, a **tree-walking interpreter** (or just **tree-walker**) evaluates the code by executing a recursive evaluation function on the AST which you can probably invent for yourself right now if you look at the diagram above for thirty seconds. (If you can't, it will be explained later.)

An **interpreter** generally is something that executes code *other than* machine code. However, some people use that term to mean only a treewalker, and would call something that executes bytecode a **virtual machine** or **VM**.

If I had to explain what makes it a "virtual machine", I would say the distinction is that since the treewalker is just a collection of recursive functions calling one another, it never explicitly has to represent *state*, because the intermediate calculations are automatically pushed onto the stack as stack frames at each function call. By contrast, the virtual machine must model its execution as the operations it executes modifying the **state** of the machine until it contains our final result.

## Note on Pipefish

I will be using Pipefish to illustrate these concepts not only because it is objectively The Bestest Language In The Whole World™, but because it shares with Python the quality of being runnable pseudocode, while being simpler and having a proper type system and having pure and referentially transparent functions and immutable values. You should have no trouble reading it in combination with the explanations of what each bit of code does.

## The lexer

We will first need a lexer to produce a string of tokens. These will consist of numbers, the operations `+`, `-` (both as infix and prefix) `*`, `/`, `^` and `!`, plus numbers `42`, `99`, etc. A number like `-42` will be analysed as the prefix `-` followed by the number `42`, as in our previous examples.

We have no need for metadata, and every need for simplicity, so we will in fact just get our tokens as a list of strings and integers. This is our lexer. This is a very unsophisticated technique wich doesn't scale, because lexers are boring and I have no interest in teaching you how to do a good one. Let's get this over with and get onto the good stuff.

```

```

## RPN

I promised you an interpreter for proper arithmetic expressions, but first, for reasons that will become clear, we'll do an **RPN** calculator.

RPN is a style of notation where we write the operator on the right-hand side of its operands. Instead of `5 + 3`, we write `5 3 +`.

The disadvantage of this is that complicated expressions become harder for humans to read than our traditional notation with infixes and parentheses. It has some advantages, though. One is that we need neither parentheses nor PEMDAS to express our intent without ambiguity: e.g. the infix expressions `(2 + 3) * 4!` and `2 + (3 * 4!)` in RPN are `2 3 + 4 ! *` and `2 3 4 ! * +` respectively.

We will however have to add an operator. In PEMDAS, we can tell the difference between `-` as a prefix and `-` as an infix. In RPN, we can't, we have to know the arity of the operator, and so we will add `~` for negation to our lexer.

An expression such as `1 +` would be **ill-formed**, since `+` requires two operands; similarly `1 2 3 +` would be ill-formed, because after we've performed the addition we still have two numbers `1 5` with nothing to do with them. A **well-formed** expression will use up its operands and operators at the same time, leaving us with just one number which will be the correct result.

## Evaluating RPN by rewriting

There are a number of ways to evaluate such an expression. One is to look (at random if you like) for any sequence where n numbers are followed by an operator of arity n, e.g. when `!` follows one number or `*` follows two numbers. Then you calculate that bit of the expression and replace it. You continue doing that until you have one number left. (The proof that this must terminate with one number on the stack if the expression is well-formed is left as an exercise for the reader.) The fact that you can evaluate things by rewriting them can be used for serious purposes, [or for silly ones](****** rofl)

## Evaluating RPN with a stack machine

The more usual way to evaluate RPN expressions, however, is with a *stack machine*.

The algorithm is this. We start at the head of our list of tokens with an empty stack, and then iteratively:

* If the head of the tokens is a number, remove it from the head and push it onto the stack.
* If the thead of the tokens is a number is a symbol of arity `n`, remove it from the head, pop `n` numbers off the stack, apply the operation to them, and push the result (necessarily another number) onto the stack. If there *aren't* `n` numbers on the stack, the expression is ill-formed and we throw an error.
* We repeat this until we run out of symbols. At that point if we have exactly one number on the stack, this is the correct answer, and if we don't have exactly one number on the stack, then the expression was ill-formed to start with.

So we can express that in code like this.

```

```

We can get Pipefish to tell us what it's doing. Let's put in ***** and see the workings of the interpreter.

```



```

Note that our stack machine makes no mention of our parenthesis tokens, because RPN doesn't need or use them. Let's move on to conventional PEMDAS notation, which does.

## A Pratt parser

There are many ways to write a parser: the Pratt parser is particuarly beautiful and simple. My presentation of it will be different from others, because if we think about and write the parser in a functional style, it becomes very simple indeed.

First, let's quickly define recursively what it means for one of our expressions to be well-formed.

* For any valid number (a sequence of the digits `0` ... `9` not beginning with `0`), that number on its own is a well-formed expression.
* If `E` and `F` are well-formed expressions, and if `p`, `i` and `s` are a prefix, and infix, and a suffix respectively, then `p E`, `E i F`, and `E s` are well-formed expressions, where in this case:
   * Our only prefix is `-`.
   * Our infixes are `+`, `-`, `*`, `/`, and `^`.
   * Our only suffix is `!`.
* If `E` is a well-formed expression, so is `(E)`.

Such definitions can be made more precise with formal descriptions such as [**Backus-Naur form**]() (**BNF**), and indeed such descriptions can then be fed into a **parser generator** which will write a parser for you. For our purposes, BNF would be a sledgehammer to crack a nut.

Now, we may divide any well-formed expression (hereafter just "expression" 'cos that's all we really care about) into an **onset** and a **coda**, where an **onset** is a well-formed expression and a coda consists either of:

* An infix followed by an expression.
* A suffix.
* Nothing at all.

It is trivially true that an expression can be analysed this way given that the coda can be empty. But our algorithm will recursively find the shortest possible onset, then look at its coda, and if its an infix followed by an expression, will analyse that into the shortest possible onset followed by its coda, etc, until the whole expression has been analysed that way.

What we mean by "the shortest possible onset" is the one that would still be well-formed if we put in parentheses to show the order of operations. For example, if we have `3 + 4 + 5`, we could group that as `3 + (4 + 5)`, and say that `3` is the onset and `+ (4 + 5)` is the coda, and we then analyse the expression `4 + 5` into the onset `4` and the coda `+ 5`.

But if we had `3 * 4 + 5`, then the parentheses would have to go like this: `(3 * 4) + 5`, and so `3 * 4` is the shortest possible onset and `+ 5` is the remaining coda; and we would then analyze `3 * 4` into an onset of `3` and a coda of `* 4`.

And the way we can tell that the onset of `3 * 4 + 5` has to be `3 * 4` and not `3` is that the precedence of `*` is higher than the precedence of `+`. That's the supposedly difficult bit of a Pratt parser. It isn't. The rule is: to find out if an infix can be the start of the coda or must be part of the onset, we compare its precedence with that of the next operator.

So a short way of summarizing the Pratt algorithm looks like this:

* Start off by representing the expression as a dummy empty placeholder AST node representing the onset, and a list of all the tokens representing the coda; and with a current precedence of 0, the lowest possible.
* Find the smallest number of tokens that *could be* the onset of the expression. These will necessarily be an expression themself, call it `E`. We will call the remaining tokens [T] `E` will be either:
   * An expression grouped by parentheses.
   * An expression consisting of a prefix and an expression.
   * A number followed by nothing.
   * A number followed by a suffix.
   * A number followed by an infix.
In the first three cases, `E` is definitely the onset, and if they're followed by an operation that's the start of the coda. We will leave suffixes alone for now: they will be easy to understand once we've done infixes.

And in the last case, that of infixes, we need to consider what precedence we were using as determined by the previous operator we used (and so we must pass this around from function to function), and of the infix we're looking at now, call it `i` = `T[0]`, to determine whether  `i` really belongs to the start of the coda or the middle of the onset.

If the precedence of `i` is no higher than the one we passed into the function, then we can treat `i` as the start of the coda.

If it *is* higher, we need to set our current precedence level to the precedence of that `i`, parse whatever's to the right of `i` (`T[1::len T]`)into an onset `F` and coda `C`, and then say that the result of this whole manouvre is that our *onset* is `E i F` and our *coda* is `C`.

This is the only "clever bit" of the Pratt parser: we get it to *wrongly* analyse the expression as a node `E`, followed by tokens `T` consisting of `i F C`, where `F` is the onset and `C` is the coda of `F C`. We then rearrange these bits and pieces to get the *right* analysis of the expression, which consists of an onset `E i F` and its coda `C`.

Let's look at the code!

```


```

Again we can ask Pipefish to give us some insight into what it's doing:

```


```

We skipped over suffixes. Now you understand what it's doing, it's very easy to see that all we need to do is change ***** to read like this:

## A treewalker

So now we can put PEMDAS expressions into an AST, we can consider writing a treewalker to get them out.

The algorithm is childishly simple. We evaluate a node as follows:
* If the node is a number, that's its value
* Otherwise it's an operator. We recursively evaluate its children, and then*to those values we apply the function appropriate to the operator to get the value, i.e. adding them together if it's `+`.

Here's the code:


Let's watch Pipefish at work again.







## RPN-ification

Even if you aren't going to pursue langdev, it may still sometimes be a useful fact that a tree structure can be serialized into a nice linear sequence of RPN, by a procedure we might call "pushing the tree gently over to the right".

To RPN-ify a node, the algorithm goes like this:

* If the node is a number, return a list containing only that number.
* If the node is an operator, recursively RPN-ify its argument(s), concatentate the resulting lists together if there's more than one, and append the operator to the list. In code:



And we'll watch Pipefish RPN-ify our usual expression.



## Writing a compiler and a VM

We've already written both of these things. All we need is to join them together.



That was easy, wasn't it?

## A Pratt compiler

However, we didn't really need to create the AST. It may help us thing about the process, but its entirely dispensable. All we need to do is change our parser so that instead of representing the onset as a node and the coda as a list of tokens in PEMDAS form, it represents the onset as a list of tokens in RPN form instead. This is very easy to do.

```

```

## A Pratt interpreter

If we were writing a full-scale programming language, what we just did might be a useful start on a bytecode interpreter. A number of real virtual machines work very like this, storing their bytecode in RPN form for rapid execution and executing it on a stack machine.

The reason the bytecode is useful in a real language is that because it can have variables in it, it can do something different each time it executes. Our calculator doesn't need to do this, and so we can dispense with any more structured representation at all of our code beyond the list of tokens, and rewrite our Pratt parser one more time into a **Pratt interpreter** which recursively analyses the tokens into a coda consisting of a list of tokens and an *integer* representing the onset.

```







```
## And that's it

One lexer, one parser, three interpreters, and two compilers, exactly as promised. I hope you all had fun. If so, please leave a star on [the Pipefish repo](). Have a nice day!


