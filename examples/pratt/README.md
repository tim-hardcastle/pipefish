Let's write a parser! (and a lexer, a prettyprinter, three interpreters, and two compilers).

## Introduction

This exercise is suitable as an introduction to some basic concepts for langdev beginners, but it may also give more experienced langdevs some things to think about. For one thing, a Pratt parser in a purely functional style is a thing of almost inconceivable simplicity and beauty; and also you may not have seen some of the tricks I can do with them.

## Terminology

Let's define some terms.

* A **lexer** (and/or **tokenizer**) takes code considered just as a string of characters, and converts it into the next higher unit of structure, the `token`, for example turning the string `"(-42 + 99) * 4!"` into the list `["(", "-", "42", "+", "99", ")", "*", "4", "!"]`.

* Normally instead of the tokens being a list of mere bare strings they have **metadata** to say where in the code they came from: in a mature language, the name of the source code file, the line number, and the position in the line. (We're not going to do this, but in a real language implementation you should tag *all* your artifacts with metadata showing where they come from: you *will* need it.)

* A **parser** takes this string of tokens and turns it into a data structure that more closely represents the syntactic structure of the code. The most usual form, and the one we will use here, is to turn the expression into an **abstract syntax tree**, or **AST**, which assigns the tokens to nodes of a tree such that the branches of a node are the arguments of an operation. For example, from the tokens `["(", "-", "42", "+", "99", ")", "*", "4", "!"]`, the AST would be:
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

If I had to explain what makes it a "virtual machine", I would say the distinction is that since the treewalker is just a collection of recursive functions calling one another, it never explicitly has to represent *state*, because the intermediate calculations are automatically pushed onto the stack as stack frames at each function call. By contrast, the virtual machine must model its execution as the operations it executes modifying the state of the machine until it contains our final result.

## Note on the code

I will be using Pipefish to illustrate these concepts not only because it is objectively The Bestest Language In The Whole World™, but because it shares with Python the quality of being runnable pseudocode, while being simpler and having a proper type system and having pure and referentially transparent functions and immutable values. You should have no trouble reading it in combination with the text and comments.

It also has the advantage that it is easy to peek inside the workings of it as it runs and see what it's up to.

Writing an interpreter/compiler allows us to structure our code very nicely with modularity and encapsulation. For convenience, we're *not* going to do that, and instead will use `include` statements to smoosh our little files together into programs that promiscuously share even their `private` functions.

All the files can be found in the [examples/pratt folder of the Pipefish repo](https://github.com/tim-hardcastle/pipefish/tree/main/examples/pratt).

## The lexer

We will first need a lexer to produce a string of tokens. These will consist of numbers, the operations `+`, `-` (both as infix and prefix) `*`, `/`, `^` and `!`, plus numbers `42`, `99`, etc. A number like `-42` will be analysed as the prefix `-` followed by the number `42`, as in our previous examples.

We have no need for metadata, and every need for simplicity, so we will just get our tokens as a list of runes (the symbols) and integers. This is our lexer.

```
const 

WHITESPACE = set(' ', '\t')
SYMBOLS = set('+', '-', '*', '/', '^', '~', '!', '(', ')')
NUMERALS = set('0', '1', '2', '3', '4', '5', '6', '7', '8', '9') 

// We wrap our recursive function with two parameters inside a non-recursive function
// with just one, so we can conveniently call it.
def

lex(code string) :
    first lexHead [], code

// And now the recursive function.
lexHead(tokens list, code string) -> list, string :
    code == "" :
        tokens, ""
    head in WHITESPACE :
        lexHead tokens, tail 
    head in SYMBOLS :
        lexHead tokens & head, tail 
    head in NUMERALS :
        lexHead tokens & number, numberTail
    else :
        error "unexpected rune `'" + string(head) + "'`"
given :    
    head = code[0]
    tail = code[1::len code]
    number, numberTail = lexNumber tokens, code
    
lexNumber(tokens, code) -> int, string :
    int(numString), tail
given :
    numString = slurpNumber(code)
    tail = code[len(numString)::len(code)]

slurpNumber(code) -> string :
    from numString = "" for _::digit = range code :
        digit in NUMERALS :
            numString & digit 
        else :
            break numString
```

## An RPN calculator

I promised you an interpreter for proper arithmetic expressions, but first, for reasons that will become clear, we'll do an RPN calculator.

**RPN** (short for **Reverse Polish Notation**) is a style of notation where we write the operator on the right-hand side of its operands. Instead of `5 + 3`, we write `5 3 +`.

The disadvantage of this is that complicated expressions become harder for humans to read than our traditional notation with infixes and parentheses. It has some advantages, thoug, and one is that we need neither parentheses nor precedence to express our intent without ambiguity: e.g. the infix expressions `(2 + 3) * 4!` and `2 + (3 * 4!)` in RPN are `2 3 + 4 ! *` and `2 3 4 ! * +` respectively.

We will however have to add an operator. In PEMDAS, we can tell the difference between `-` as a prefix and `-` as an infix. In RPN, we can't, we have to know the arity of the operator, and so we will add `~` for negation to our lexer.

An expression such as `1 +` would be **ill-formed**, since `+` requires two operands; similarly `1 2 3 +` would be ill-formed, because after we've performed the addition we still have two numbers `1 5` with nothing to do with them. A **well-formed** expression will use up its operands and operators at the same time, leaving us with just one number on the stack which will be the correct result.

## Evaluating RPN by rewriting

There are a number of ways to evaluate such an expression. One is to look (at random if you like) for any sequence where n numbers are followed by an operator of arity n, e.g. when `!` follows one number or `*` follows two numbers. Then you calculate that bit of the expression and replace it. You continue doing that until you have one number left. (The proof that this must terminate with one number on the stack if the expression is well-formed is left as an exercise for the reader.) The fact that you can evaluate things by rewriting them can be used for serious purposes, [or for silly ones](https://github.com/tim-hardcastle/pipefish/tree/main/examples/rofl).

## Evaluating RPN with a stack machine

The more usual way to evaluate RPN expressions, however, is with a *stack machine*.

The algorithm is this. We start at the head of our list of tokens with an empty stack, and then iteratively:

* If the head of the tokens is a number, remove it from the head and push it onto the stack.
* If the head of the tokens is a number is a symbol of arity `n`, remove it from the head, pop `n` numbers off the stack, apply the operation to them, and push the result (necessarily another number) onto the stack. If there *aren't* `n` numbers on the stack, the expression is ill-formed and we throw an error.
* We repeat this until we run out of symbols. At that point if we have exactly one number on the stack, this is the correct answer, and if we don't have exactly one number on the stack, then the expression was ill-formed to start with.

This description of the algorithm should give you some insight into the advantages of RPN. There is a sense in which it is in the "right" order, in that we obviously need to put our values into the memory of our machine before we can apply an operation to them. Given the RPN form, we can just move linearly along the tokens processing them one at a time. To emphasize this point, I will write the main body of this algorithm using a `for` loop (a [pure, referentially transparent `for` loop in which all the variables are immutable](https://github.com/tim-hardcastle/Pipefish/wiki/For-loops)).

```
include

"lexer.pf"  
"mathfns.pf"  

newtype 

~~ The state of the VM: a stack of integers, and a list of tokens still to be processed.
State = struct(stack, tokens list)

~~ This will contain information about what a given operator does, as a value in the
~~ `RPN_INFO` map below.
Info = struct(arity int, fn func)

const

RPN_INFO = map('+'::Info(2, func(L) : L[0] + L[1]),
                 .. '-'::Info(2, func(L) : L[0] - L[1]),
                 .. '*'::Info(2, func(L) : L[0] * L[1]),
                 .. '/'::Info(2, func(L) : L[0] div L[1]),
                 .. '^'::Info(2, func(L) : exp(L[0], L[1])),
                 .. '~'::Info(1, func(L) : - L[0]),
                 .. '!'::Info(1, func(L) : fac(L[0])),)

def

~~ Executes code given as a string in RPN form.
exec(code string) :
    code -> lex -> run(State([], that)) -> that[stack][0]

private

run(initial State) -> State :
    from S = initial for S[tokens] != [] :
        head in int :
            State(S[stack] & head, tail)
        else :
            State(poppedStack & newHead, tail)
    given :
        head = S[tokens][0]
        tail = S[tokens][1::len S[tokens]]
        info = RPN_INFO[head]
        poppedStack = S[stack][0::len(S[stack])-info[arity]]
        F = info[fn]
        args = S[stack][(len(S[stack])-info[arity])::len(S[stack])]
        newHead = F(args)
```

We can get Pipefish to tell us what it's doing. Let's put in `exec "2 3 * 4 +"` and see the workings of the main loop as it goes round.

```
▪ We called function run (defined at line 33) with initialState = 
    State([], [2, 3, '*', 4, '+']). 
  ▪ We entered the loop at line 34 with S = State([], [2, 3, '*', 4, '+']). 
  ▪ At line 35 we evaluated the condition head in int. 
  ▪ The condition succeeded. 
  ▪ At line 36 the body of the for loop evaluated to State([2], [3, '*', 4, '+']). 
  ▪ We entered the loop at line 34 with S = State([2], [3, '*', 4, '+']). 
  ▪ At line 35 we evaluated the condition head in int. 
  ▪ The condition succeeded. 
  ▪ At line 36 the body of the for loop evaluated to State([2, 3], ['*', 4, '+']). 
  ▪ We entered the loop at line 34 with S = State([2, 3], ['*', 4, '+']). 
  ▪ At line 35 we evaluated the condition head in int. 
  ▪ The condition failed. 
  ▪ At line 37 we took the else branch. 
  ▪ At line 38 the body of the for loop evaluated to State([6], [4, '+']). 
  ▪ We entered the loop at line 34 with S = State([6], [4, '+']). 
  ▪ At line 35 we evaluated the condition head in int. 
  ▪ The condition succeeded. 
  ▪ At line 36 the body of the for loop evaluated to State([6, 4], ['+']). 
  ▪ We entered the loop at line 34 with S = State([6, 4], ['+']). 
  ▪ At line 35 we evaluated the condition head in int. 
  ▪ The condition failed. 
  ▪ At line 37 we took the else branch. 
  ▪ At line 38 the body of the for loop evaluated to State([10], []). 
```

Note that our stack machine makes no mention of our parenthesis tokens, because RPN doesn't need or use them. Let's move on to conventional PEMDAS notation, which does.

## A Pratt parser

There are many ways to write a parser: the Pratt parser is particularly beautiful and simple.

First, let's quickly define recursively what it means for one of our PEMDAS expressions to be well-formed.

* For any valid number (a sequence of the digits `0` ... `9` not beginning with `0`), that number on its own is a well-formed expression.
* If `E` and `F` are well-formed expressions, and if `p`, `i` and `s` are a prefix, and infix, and a suffix respectively, then `p E`, `E i F`, and `E s` are well-formed expressions.
* If `E` is a well-formed expression, so is `(E)`.

In our explanation of how a Pratt parser works, we will assume that the expression we're working on is well-formed: the actual code will throw various errors if it isn't.

So, to parse a list of tokens into an AST, we recursively do this:

* We look at the start of the list of tokens, the smallest number of tokens which, taken on its own would make a well-formed expression, and we turn that into a node. Given our definitions above, this will either consist of:

  * A number.
  * A prefix expression.
  * An expression grouped by parentheses.

* We then look at the rest of the list of tokens to see what we should do next, where, from the definition of a well-formed expression, the rest of the tokens must consist of nothing at all (we've finished parsing and should return) or an infix or suffix operator which will have to be consumed eventually.

The crucial thing to note is this. Suppose we have something like `2 * 3 + 4`. Then if we just naively analyzed it as a node `2`, an infix expression `*`, and a list of tokens `3 + 4`, and then parsed the rest of the tokens into a node and joined the result together using the infix, we'd get  a node representing `2 * (3 + 4)`, which is a different expression.

What we do instead is recursively parse the remainder of the tokens just enough to turn that too into a node and a shorter tail of tokens, and then we have a node `2` and the infix `*`, and a node `3`, and a remaining list of tokens `+ 4`.

Now we can join `2` together with `3` by the infix to get a binary node `2 * 3`, and continue with the algorithm.

The algorithm decides when it needs to do perform this trick (call it the "Pratt manoeuvre") and when it can just proceed naively, according to whether the precedence of the operator is strictly higher than the precedence of the last operator we looked at (starting with this set at 0). So in the example above, we start with the precedence at 0, see that `*` has a higher precedence than 0, carry out the Pratt manoeuvre, see that `+` has a lower precedence than `*`, and proceed naively, returning from our recursive function calls until we're looking at the tail of the tokens with a precedence of 0 again.

If on the other hand our expression was `2 * 3 ^ 4`, then when the parser arrives at the `^` it will see that this has a higher precedence than `*` and will perform the Pratt manoeuvre again.

Let's express this as code.

```
~~ Simple Pratt parser for arithmetic expressions.

include

"lexer.pf"

newtype 

// We declare the nodes out of which we will build our AST.

NumberNode = struct(value int)
PrefixNode = struct(op rune, arg Node)
InfixNode = struct(op rune, leftArg, rightArg Node)
SuffixNode = struct(op rune, arg Node)
EmptyNode = struct()

Node = abstract NumberNode/PrefixNode/InfixNode/SuffixNode/EmptyNode

const 

~~ Information about the precedence of operations.
// The great thing about the Pratt parser is that what in other parsers would rely
// on a bunch of function calls and `if` statements and switches can be summarized
// in a data structure. This is it.
INFO = map(..
    .. PrefixNode::map(..
        .. '-'::4,
        .. ),
    .. InfixNode::map(..
        .. '+'::1,
        .. '-'::1,
        .. '*'::2,
        .. '/'::2,
        .. '^'::3,
        .. ),
    .. SuffixNode::map(..
        .. '!'::5,
        .. ),
    ..)

def

~~ Parses a string into an AST.
// We need to wrap the recursive `parseExpression` function in this non-recursive function so
// that we can recognize if/when there are still tokens left over after the recursion terminates.
// This function knows where it is in the call tree and no instance of `parseExpression` can.
parse(code string) -> Node :
    len leftovers > 0 :
        error "unexpected `" + string(head) + "`"
    else :
        result 
given :
    result, leftovers = code -> lex -> parseExpression(that, 0)
    head = leftovers[0]

private

parseExpression(tokens list, prec int) -> Node, list : 
    parseStart(tokens) -> parseRest(that[0], that[1], prec)

// The start of an expression will be either :
//   (A) a prefix followed by an expression, followed by the rest of the expression.
//   (B) an expression in parentheses, followed by the rest of the expression
//   (C) a number followed by the rest of the expression.
//   (D) an infix or `)`, *if* the expression is ill-formed.
//
parseStart(tokens list) -> Node, list :  
    head in keys INFO[PrefixNode] :
        parsePrefixExpression(tokens)
    head in rune and head == '(' :
        parseGroupedExpression(tail)
    head in int : 
        NumberNode(head), tail  
    else :
        error "unexpected token `" + string(head) + "`"
given :
    head = tokens[0]
    tail = tokens[1::len tokens]

// We parse the various cases in the branches of parseStart.

parsePrefixExpression(tokens list) -> Node, list: 
    PrefixNode(head, rightNode), newTail 
given :
    head = tokens[0]
    tail = tokens[1::len tokens]
    rightNode, newTail = parseExpression(tail, INFO[PrefixNode][head])

parseGroupedExpression(tokens list) -> Node, list : 
    shouldBeRparen == ')' :
        node, newTail 
    else : 
        error "expected `)`"
given :
    node, rparenAndNewTail = parseExpression(tokens, 0)
    shouldBeRparen = rparenAndNewTail[0]
    newTail = rparenAndNewTail[1::len rparenAndNewTail]

parseRest(leftNode Node, tokens list, prec int) -> Node, list : 
    valid headPrec and headPrec > prec :
        parseRest(InfixNode(head, leftNode, rightNode), newTail, prec) 
    valid suffixPrec and suffixPrec > prec :
        parseRest(SuffixNode(head, leftNode), tail, prec)
    else : 
        leftNode, tokens
given :
    head = tokens[0]
    tail = tokens[1::len tokens]
    headPrec = INFO[InfixNode][head]
    suffixPrec = INFO[SuffixNode][head]
    rightNode, newTail = parseExpression(tail, headPrec)
```

Again we can ask Pipefish to talk us through what it's doing if we do `parse "2 * 3 + 4"`.

```
  ▪ We called function parseExpression (defined at line 58) with tokens = 
    [2, '*', 3, '+', 4], prec = 0. 
  ▪ We called function parseStart (defined at line 67) with tokens = [2, '*', 3, '+', 4]
    . 
  ▪ At line 68 we evaluated the condition head in keys INFO[PrefixNode]. 
  ▪ The condition failed. 
  ▪ At line 70 we evaluated the condition head in rune and head == '('. 
  ▪ The condition failed. 
  ▪ At line 72 we evaluated the condition head in int. 
  ▪ The condition succeeded. 
  ▪ At line 73 function parseStart returned (NumberNode(2), ['*', 3, '+', 4]). 
  ▪ We called function parseRest (defined at line 99) with leftNode = NumberNode(2), 
    tokens = ['*', 3, '+', 4], prec = 0. 
  ▪ At line 100 we evaluated the condition valid headPrec and headPrec > prec. 
  ▪ The condition succeeded. 
  ▪ We called function parseExpression (defined at line 58) with tokens = [3, '+', 4], 
    prec = 2. 
  ▪ We called function parseStart (defined at line 67) with tokens = [3, '+', 4]. 
  ▪ At line 68 we evaluated the condition head in keys INFO[PrefixNode]. 
  ▪ The condition failed. 
  ▪ At line 70 we evaluated the condition head in rune and head == '('. 
  ▪ The condition failed. 
  ▪ At line 72 we evaluated the condition head in int. 
  ▪ The condition succeeded. 
  ▪ At line 73 function parseStart returned (NumberNode(3), ['+', 4]). 
  ▪ We called function parseRest (defined at line 99) with leftNode = NumberNode(3), 
    tokens = ['+', 4], prec = 2. 
  ▪ At line 100 we evaluated the condition valid headPrec and headPrec > prec. 
  ▪ The condition failed. 
  ▪ At line 102 we evaluated the condition valid suffixPrec and suffixPrec > prec. 
  ▪ The condition failed. 
  ▪ At line 104 we took the else branch. 
  ▪ At line 105 function parseRest returned (NumberNode(3), ['+', 4]). 
  ▪ At line 59 function parseExpression returned (NumberNode(3), ['+', 4]). 
  ▪ We called function parseRest (defined at line 99) with leftNode = 
    InfixNode('*', NumberNode(2), NumberNode(3)), tokens = ['+', 4], prec = 0. 
  ▪ At line 100 we evaluated the condition valid headPrec and headPrec > prec. 
  ▪ The condition succeeded. 
  ▪ We called function parseExpression (defined at line 58) with tokens = [4], prec = 1. 
  ▪ We called function parseStart (defined at line 67) with tokens = [4]. 
  ▪ At line 68 we evaluated the condition head in keys INFO[PrefixNode]. 
  ▪ The condition failed. 
  ▪ At line 70 we evaluated the condition head in rune and head == '('. 
  ▪ The condition failed. 
  ▪ At line 72 we evaluated the condition head in int. 
  ▪ The condition succeeded. 
  ▪ At line 73 function parseStart returned (NumberNode(4), []). 
  ▪ We called function parseRest (defined at line 99) with leftNode = NumberNode(4), 
    tokens = [], prec = 1. 
  ▪ At line 100 we evaluated the condition valid headPrec and headPrec > prec. 
  ▪ The condition failed. 
  ▪ At line 102 we evaluated the condition valid suffixPrec and suffixPrec > prec. 
  ▪ The condition failed. 
  ▪ At line 104 we took the else branch. 
  ▪ At line 105 function parseRest returned (NumberNode(4), []). 
  ▪ At line 59 function parseExpression returned (NumberNode(4), []). 
  ▪ We called function parseRest (defined at line 99) with leftNode = 
    InfixNode('+', InfixNode('*', NumberNode(2), NumberNode(3)), NumberNode(4)), tokens = 
    [], prec = 0. 
  ▪ At line 100 we evaluated the condition valid headPrec and headPrec > prec. 
  ▪ The condition failed. 
  ▪ At line 102 we evaluated the condition valid suffixPrec and suffixPrec > prec. 
  ▪ The condition failed. 
  ▪ At line 104 we took the else branch. 
  ▪ At line 105 function parseRest returned 
    (InfixNode('+', InfixNode('*', NumberNode(2), NumberNode(3)), NumberNode(4)), []). 
  ▪ At line 101 function parseRest returned 
    (InfixNode('+', InfixNode('*', NumberNode(2), NumberNode(3)), NumberNode(4)), []). 
  ▪ At line 101 function parseRest returned 
    (InfixNode('+', InfixNode('*', NumberNode(2), NumberNode(3)), NumberNode(4)), []). 
  ▪ At line 59 function parseExpression returned 
    (InfixNode('+', InfixNode('*', NumberNode(2), NumberNode(3)), NumberNode(4)), []). 
```

## A treewalker

So now we can put PEMDAS expressions into an AST, we can consider writing a treewalker to evaluate them.

The algorithm is childishly simple. We evaluate a node as follows:
* If the node is a number, that's its value
* Otherwise it's an operator. We recursively evaluate its children, and then*to those values we apply the function appropriate to the operator to get the value, i.e. adding them together if it's `+`.

Here's the code:

```
include 

"mathfns.pf"
"parser.pf"

const 

OPS = map(..
    .. PrefixNode::map(..
        .. '-'::(func(x int) : -x),
        .. ),
    .. InfixNode::map(..
        .. '+'::(func(x, y int) : x + y),
        .. '-'::(func(x, y int) : x - y),
        .. '*'::(func(x, y int) : x * y),
        .. '/'::(func(x, y int) : x div y),
        .. '^'::(func(x, y int) : exp(x, y)),
        .. ),
    .. SuffixNode::map(..
        .. '!'::(func(x int) : fac(x)),
        .. ),
..)

def

ev(code string) :
    code -> parse -> walk

private

walk(n Node) -> int :
    n in NumberNode :
        n[value]
    n in InfixNode :
        fnForOperation(walk(n[leftArg]), walk(n[rightArg]))
    else :
        fnForOperation(walk(n[arg]))
given :
    fnForOperation = OPS[type n][n[op]]
```

Let's watch Pipefish at work again, as we do `ev "2 * 3 + 4"`.

```
  ▪ We called function walk (defined at line 31) with n = 
    InfixNode('+', InfixNode('*', NumberNode(2), NumberNode(3)), NumberNode(4)). 
  ▪ At line 32 we evaluated the condition n in NumberNode. 
  ▪ The condition failed. 
  ▪ At line 34 we evaluated the condition n in InfixNode. 
  ▪ The condition succeeded. 
  ▪ We called function walk (defined at line 31) with n = 
    InfixNode('*', NumberNode(2), NumberNode(3)). 
  ▪ At line 32 we evaluated the condition n in NumberNode. 
  ▪ The condition failed. 
  ▪ At line 34 we evaluated the condition n in InfixNode. 
  ▪ The condition succeeded. 
  ▪ We called function walk (defined at line 31) with n = NumberNode(2). 
  ▪ At line 32 we evaluated the condition n in NumberNode. 
  ▪ The condition succeeded. 
  ▪ At line 33 function walk returned 2. 
  ▪ We called function walk (defined at line 31) with n = NumberNode(3). 
  ▪ At line 32 we evaluated the condition n in NumberNode. 
  ▪ The condition succeeded. 
  ▪ At line 33 function walk returned 3. 
  ▪ At line 35 function walk returned 6. 
  ▪ We called function walk (defined at line 31) with n = NumberNode(4). 
  ▪ At line 32 we evaluated the condition n in NumberNode. 
  ▪ The condition succeeded. 
  ▪ At line 33 function walk returned 4. 
  ▪ At line 35 function walk returned 10. 
```

## A pretty-printer

We can always ugly-print our AST by putting parentheses around every operator and its arguments, for example turning `(-42 + 99) * 4!` into `(((-42) + 99) * (4!))`. For the purposes of debugging your parser and seeing where it's going wrong, this may be the most useful form: for other purposes it's ugly and confusing.

The rule we need is that each operator should put parentheses around any of its child nodes that has a lower operator than it does, as follows:

```
~~ Prettyprinter.

include 

"parser.pf"

def 

~~ Prettyprints the given node.
print(n Node) -> string :
    ppr(n, 0) 

private

ppr(n Node, prec int) : 
    n in NumberNode :    
        string n[value]   
    prec <= INFO[type n][n[op]] :   
        describe n, INFO[type n][n[op]]   
    else :   
        "(" + (describe n, INFO[type n][n[op]]) + ")" 

describe(n InfixNode, prec int) -> string : 
    ppr(n[leftArg], newPrec) + " " + string(n[op]) + " " + ppr(n[rightArg], newPrec) 
given :
    newPrec = INFO[InfixNode][n[op]] 

describe(n PrefixNode, prec int) -> string : 
    n[op] & ppr(n[arg], newPrec) 
given :
    newPrec = INFO[PrefixNode][n[op]] 

describe(n SuffixNode, prec int) -> string : 
    ppr(n[arg], newPrec) & n[op] 
given :
    newPrec = INFO[SuffixNode][n[op]] 
```

## RPN-ification

Even if you aren't going to pursue langdev, it is still sometimes a useful fact that a tree structure can be serialized into a nice linear sequence of RPN, by a procedure we might call "pushing the tree gently over to the right".

To RPN-ify a node, the algorithm goes like this:

* If the node is a number, return a list containing only that number.
* If the node is an operator, recursively RPN-ify its argument(s), concatenate the resulting lists together if there's more than one, and append the operator to the list. In code:

```
include

"parser.pf"

def 

rpnf(n Node) -> list :
    n in NumberNode :
        [n[value]]
    n in InfixNode : 
        rpnf(n[leftArg]) + rpnf(n[rightArg]) & n[op]
    else :
        rpnf(n[arg]) & tweakOp
given :
    tweakOp = (n in PrefixNode and n[op] == '-' : '~' ; else : n[op])
```

## Writing a compiler and a VM

We've already written both of these things. All we need is to join them together.

```
include 

"parser.pf"
"rpn.pf"
"rpnify.pf"

def 

~~ Evaluates a string in PEMDAS form, returning an integer.
ev(code string) -> int :
    code -> compile -> run(State([], that)) -> that[stack][0]

~~ Compiles a string in PEMDAS form to a list in RPN form.
compile(code string) -> list :
    code -> parse -> rpnf
```

That was easy, wasn't it?

Now, what was the point of that? Well, the RPN version is *faster*. Instead of having to make a bunch of recursive calls and returns to walk around the tree, we're now using a `for` loop to iterate through a list. For real languages, the speed-up in execution is trival, and the implementation of a VM is not particularly challenging.

## A Pratt compiler

However, now we're not walking the tree, we didn't really need to create the AST. It may help us to think about the process of compilation, but it's not essential. All we need to do is change our parser so that instead of constructing an AST, it outputs a list of tokens in RPN form. This requires some minimal systematic changes to the code.

```
include 

"parser.pf"
"rpn.pf"
"rpnify.pf"

def 

~~ Evaluates a string in PEMDAS form, returning an integer.
ev(code string) -> int :
    code -> compile -> run(State([], that)) -> that[stack][0]

~~ Compiles a string in PEMDAS form to a list in RPN form.
compile(code string) -> list :
    code -> parse -> rpnf
```

## A Pratt interpreter

However, since we're just writing a calculator, what we did was kind of pointless. Bytecode is useful because we can execute it again and again with different variables, having compiled it only once. If we do that with arithmetic expressions, we'll get the same answer each time, so we don't need RPN bytecode as an artifact.

And so we can dispense any more structured representation at all of our code beyond the original list of tokens, and rewrite our Pratt parser one final time into a **Pratt interpreter** which recursively analyses the tokens into an integer representing the intermediate result and a list of tokens still to be processed.

By this point you can probably imagine what the code looks like, but here it is for completeness.

```
~~ Pratt interpreter for evaluating arithmetic expressions in PEMDAS form.
// This is so similar to the parser and the second compiler that most of the refactoring was 
// done by search-and-replace, and so the comments on this are minimal.

include

// We will re-use the `OPS` map in the treewalker, and the map of precedences from the parser
// (which the treewalker already includes).
"treew.pf"

def

~~ Evaluates a string in PEMDAS form into an integer.
evaluate(code string) -> int :
    len leftovers > 0 :
        error "unexpected `" + string(head) + "`"
    else :
        result 
given :
    result, leftovers = code -> lex -> evaluateExpression(that, 0)
    head = leftovers[0]

private

evaluateExpression(tokens list, prec int) -> int, list : 
    evaluateStart(tokens) -> evaluateRest(that[0], that[1], prec)

evaluateStart(tokens list) -> int, list :  
    head in keys INFO[PrefixNode] :
        evaluatePrefixExpression(tokens)
    head in rune and head == '(' :
        evaluateGroupedExpression(tail)
    head in int : 
        head, tail  
    else :
        error "unexpected token `" + string(head) + "`"
given :
    head = tokens[0]
    tail = tokens[1::len tokens]

// We evaluate the various cases in the branches of evaluateStart.

evaluatePrefixExpression(tokens list) -> int, list: 
    P(rightVal), newTail 
given :
    head = tokens[0]
    tail = tokens[1::len tokens]
    P = OPS[PrefixNode][head]
    rightVal, newTail = evaluateExpression(tail, INFO[PrefixNode][head])

evaluateGroupedExpression(tokens list) -> int, list : 
    shouldBeRparen == ')' :
        val, newTail 
    else : 
        error "expected `)`"
given :
    val, rparenAndNewTail = evaluateExpression(tokens, 0)
    shouldBeRparen = rparenAndNewTail[0]
    newTail = rparenAndNewTail[1::len rparenAndNewTail]

evaluateRest(leftVal int, tokens list, prec int) -> int, list : 
    valid headPrec and headPrec > prec :
        evaluateRest(I(leftVal, rightVal), newTail, prec) 
    valid suffixPrec and suffixPrec > prec :
        evaluateRest(S(leftVal), tail, prec)
    else : 
        leftVal, tokens
given :
    head = tokens[0]
    tail = tokens[1::len tokens]
    headPrec = INFO[InfixNode][head]
    suffixPrec = INFO[SuffixNode][head]
    I = OPS[InfixNode][head]
    S = OPS[SuffixNode][head]
    rightVal, newTail = evaluateExpression(tail, headPrec)
```

## And that's it

One parser, one lexer, one prettyprinter, three interpreters, and two compilers, exactly as promised. I hope you had fun. If so, please leave a star on the repo. Have a nice day!
