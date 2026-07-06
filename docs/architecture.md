# Pipefish architecture and workflow

This is a high-level overview how Pipefish works, where by high-level I mean that I won't go into the implementation of specific language features.

## Basic architecture

The implementation targets a custom VM for running Pipefish, written in Go.

Pipefish is typically meant to be used in a highly declarative manner, in which the client/user queries a Pipefish service in Pipefish via the Pipefish REPL, or via HTTP, just as one queries a SQL database in SQL.

This means that the lexer-parser-compiler-vm chain needs to be present at runtime. But the ability to declare new named functions or global variabes or data types or to import new modules, etc, should *not* be present at runtime; rather are declared in the script defining the service, and are fixed when we initialize the script.

So we need one more basic component: the initializer, which:

(1) Initializes the parser and compiler and VM according to the script.
(2) Wraps the resulting compiler-parser-VM system up in a `Service` struct that limits interaction with the service to the things you're meant to do to it at runtime.
(3) Gets thrown away along with all the data structures it contains, which we only need during initialization.

For each namespace declared for an import or external service, the initializer spawns a new initializer with a new compiler and parser, but compiling bytecode to the same VM.

The separate initializers, compilers, and parsers share information via (respectively) a `CommonIntitializerBindle`, a `CommonCompilerBindle`, and a `CommonParserBindle`, which are initialized when we initialize the root module of the service and then modified and handed down through the tree of modules.

Whether it's the `Service` struct or the `Initializer` acting on the lexer-parser-compiler-VM, the compiler is in charge of the rest of them --- it queries the parser, and writes to the VM:

During initialization :              
```
       Initializer
      /           \
     /             \
 Initializers       \
of other modules     \
   and their          \
  dependencies         \
            \      Compiler
             \    /        \
              \  /          \
               VM         Parser
                            |
                          Lexer
```
After initialization:
```
        Service
           |
        Compiler
      /    |     \
     /     |      \
    /      |       \       
   /   Compilers    Parser 
  /     of other         \
 /  modules and their   Lexer
 |    dependencies
 |   / 
  VM                         
```
## The initializer workflow

### Recursion

If there are things to be imported into namespaces, the intializer starts off by starting up other initializers, one for each namespace, and calling a method on each which will get them partway through their intitialization, including of course starting up the initializers for their namespaces, if they have any, and so on recursively, depth-first. It then calls the same method on itself, so they're all in the same point in compilation; and then we do the same thing again a few more times until initialization is finished, recursively calling another method on all the initializers depth-first to move them on to a different phase of compilation.

`NULL` imports are thrown into the namespace importing them, and built-in types and functions are imported.

The builtins come from a builtins.pf file which declares them as functions with appropriate signatures and with bodies saying that they're builtins:

```
(x float) + (y float) -> float : builtin "add_floats"
(x int) + (y int) -> int : builtin "add_integers"
(x list) + (y list) -> list : builtin "add_lists"
.
.
.
[etc]
```
This means that right up until the last moment (i.e. when compiling a function call) our workflow is just the same for builtins as for normal functions; in particular it means they can be overloaded just as easily as normal functions, which would not be the case if they were hardwired into the compiler.

### External services

At this stage we also deal with any external Pipefish services we want to use. The initializer sends an HTTP request to these for their API, which arrives in a special reverse Polish form to avoid injection. This is then used to generate a "stub" consisting of type declarations matching the API, and of function declarations with matching signatures, but with bodies that just say: "Call the external service and get the value from that". This stub is then treated just like an import, being lexed, parsed, compiled, and given a namespace.

### The lexing stage

So now let's look at one pass through one initializer. We begin with lexing, where the initializer tells the parser to tell the lexer to lex everything in the namespace.

The lexer is a fairly normal lexer, turning the source code into a stream of tokens classified as identifiers, string literals, integer literals, etc, with metadata about their orgin. However, to deal with syntactic whitespace and other challenges of the syntax, it can emit any number of tokens at a time.

This output is then passed on to the "relexer", which on an assembly-line principle performs a series of tweaks on the raw stream of tokens to make it more suitable for the parser. It also performs a few sanity checks on the data.

### The chunking stage

We then take the stream of tokens (now supplied one at a time) and put them into chunks, one for each declaration, whether of a function, command, struct type, interface import, external sevice, etc.

The resulting chunks of tokens all satisfy the `TokenizedChunk` interface, but internally are not just lists of tokens. Rather, we take this opportunity to (for example) analyze a function into its signature and its body, and the signature into identifiers, parameter names, and parameter types. This allows us to do some basic validation and see that the declarations are at least minimally well-formed.

### Adding boilerplate

We need to add some boilerplate commands to supplement any user-defined commands that have reference variables, for Reasons, and the most cromulent way to do that is to generate them at this point from the user's code in tokenized and chunked form.

### Initializing function names

Before we can parse the tokenized chunks, we need to declare to the parser the existence and syntactic role of the various things we've defined, so that e.g. it knows that the name of a function is the name of a function. Also the initializer creates a structure in the parser called a `BlingManager` which keeps track of the fancy syntax when parsing.

### Initializing types, part A

We also need to tell the parser the names of the types, since it will be parsing type expressions.

At the same time as we declare the types in the parser, we create and partly populate them in the compiler and VM: the fundamental representation of what a (concrete) type is is a number of `value.ValueType` which indexes the `ConcreteTypeInfo` field of the VM.

This is only part A of setting up the type system; there will also be parts B and C. The reason this is such a long and fragmented process is that types have to be defined in terms of other types: structs have the types of their fields defined in terms of their types, parameterized types can take types as their parameters; an interface type is a union of all types *in any module* that fit the interface, etc, etc --- so we need to *partially* define one type just enough that we can *partially* define another, and so on until we've lifted ourselves up by our bootstraps.

To try and keep this a high-level view I won't go into detail of what exactly we're creating where in each phase of type initialization.

### Parsing

We can now parse everything that needs parsing. This produces a `ParsedChunk` for each `TokenizedChunk` complex enough to warrant it (e.g. functions are parsed, import declarations can just stay as tokens). Again, like `TokenizedChunk`, `ParsedChunk` is an interface behind which the resulting code is structured according to what we're declaring. E.g. the functions are still divided up into signature and body, and the signature into identifiers and parameters and parameter types. 

The parser is a standard Pratt parser, with a few modifications to deal with user-defined infixes, postfixes, and mixfixes, i.e. the `BlingManager` the initializer set up when defining the function names.

Apart from that it's very basic because no type-checking or constant-folding or anything like that is done in the parsing stage, it's all shuffled off to the compiler.

A separate little parser deals with expressions involving types, which the parser represents as their own kind of AST.

The stages so far are performed on every module of the source code by depth-first recursion before moving on to the next step.

### Initializing types, part B

We now continue creating the types in the compiler and registering them in the VM. This is done in a number of phases each of which is performed recursively depth-first on each module of the source code before moving on the the next phase.

At the end of all this, we end up with a type number for each concrete type.

In the compiler, we will have a map associating the name of each concrete type with its type number; and a map associating the name of each abstract type with a list of the type numbers of the concrete types it contains.

In the VM, we will have information about each type sufficient for three purposes.

* To know what to do with values of that type at runtime. If for example we try to index them, or cast it to a string, is this even possible? Can we put something of type number `28` into field number `3` of type `35`?
* To know how to turn values of that type into strings and literals. When for example we want to turn a struct into a string, the VM must know the names of its fields.
* To know how to describe the structure of the types for people who import the `reflect` package.

At a later stage we will add one more piece of information for each type:

* Go interop: how do we automagically turn a Pipefish value of a given type into a Go value and back again?

## The `FunctionMap` and `FunctionForest`

We put the parsed functions into a temporary structure in the initializer (the `FunctionMap`), which groups overloaded functions together, sorted in order of the specificity of their parameter types. (Which is why we're only doing this now, up until we performed the previous step we didn't know which abstract types contain which concrete types.)

The builtin functions and type constructors have also been thrown into this table, since their signatures are just like those of normal functions.

We then convert the `FunctionTable` into a `FunctionForest` in the compiler, which we will use to perform multiple dispatch.

A `FunctionForest`, as the name suggest, consists of `Function Tree`s, one for each overloaaded function name. A `FunctionTree` is a specialized data structure, a *non-backtracking tree*, which allows us to work our way along the types of the parameters of an overloaded function from left to right, making a decision at each point, and end up in the right place without having to backtrack along the tree.

A simple example should make this clear. Suppose we have two functions:
```
foo(x any, y rune) :
    "foo 1"

foo(x int, y int) :
    "foo 2"
```
... then we wish our decision tree to embody the following logic (here I am using Pipefish as pseudocode, there's no stage in the actual compilation where Pipefish code like this is actually generated):
```
type x == int :
	type y == int :
		"foo 2"
	type y == rune :
		"foo 1"
	else :
		error "no such implementation of foo"
else :
	type y == rune :
		"foo 1"
	else :
		error "no such implementation of foo"
```
The use we make of this will be explained when we describe the compilation stage.

### Intializing types, part C

We do a little finishing up of setting up the type system. There, it's over!

### Compiling any embedded Go

Before we compile the Pipefish, we deal with any embedded Go. This is directed by the `goHandler` in the intializer module. First it puts the relevant bits of code together into a `goBucket`, the contents of which are indexed by the source file. (We emit one Go file per source file of the Pipefish code, rather than per module, for boring reasons).

The intitializer uses this data to write Go source code which declares suitable functions and types. (Any Pipefish type mentioned in a Golang function signature is automagically declared in Go.) The generated code also contains a couple of maps called `PIPEFISH_FUNCTION_CONVERTER` and `PIPEFISH_VALUE_CONVERTER` which are needed for the VM to convert types.

The initializer then uses the Go compiler to compile the generated code into an `.so` file. (If the source code hasn't been changed since last compilation, it omits the preceding steps, and uses the old `.so` file, which will still be fresh.)

Using Go's notorious `plugin` library, it slurps the function definitions and our the conversion maps out of the `.so` file, and uses them to intialize data structures in the VM that will allow it at runtime to call Go functions, and convert values from Pipefish to Go and back again.

### Topological sort

The initializer now does a topological sort on the declarations of global variables, constants, commands, and functions. This allows it to detect forbidden dependencies (e.g. a function calling a command); to detect groups of functions which may call one another recursively; and to compile functions in order of which depends on which, so that when it compiles a function it already knows the return types of the functions it calls.

### Compiling the Pipefish code

The initializer uses the compiler to compile the bodies of Pipefish functions and commands and typechecks. This works just the same as if we were compiling something typed in the REPL, except for a little extra care we have to take about interface types.

We will deal with the compiler and VM further down.

### Writing the APIs

The initializer then writes two descriptions of the API of each module, one to describe it to humans, and another in RPN to describe it to other Pipefish services.

### Returning a Pipefish service

We can then throw away the initializer and wrap the compiler in a `Service` object, to be used either through the Pipefish TUI or embedded in Go via the `pf` library.

At runtime, the `Service` object can feed a line of code to the compiler, which passes it to the parser, which returns an AST, which the compiler then compiles to the VM and executes. The compiler then rolls back the VM to its previous state (i.e. removing the code it just compiled) and returns the returned value.

## The VM

Before we move on to the compiler, we should give a description of what it's targeting.

The VM is highly specialized to be good for implementing Pipefish in. Some of the opcodes do a lot of heavy lifting --- calling an external service and deserializing the result, for example, is a single opcode.

It works on an "infinite memory" model: it has a virtual memory of arbitrary size, consisting of a list of Pipefish values.

I will refer to the bytecode as having *addresses* and the memory as having *locations*.

The bytecode consists of an 8-bit opcode and some (or no) 32-bit operands the number of which depends on the opcode. Currently they aren't sequenced in memeory as bytes but just contained in a list of structs each containing one opcode and the relevant operands. I will revisit their arrangement in memory when I think about optimization.

The operands usually, but not exclusively, refer to locations in the virtual memory. Some of them (depending on the opcode) refer to addresses in the bytecode, for when we want to jump around in it, and others, depending on the context, can refer to one of a number of useful things stashed away in the VM at compile-time: they can index a list of tokens to produce a properly-attributed runtime error; they can index a list of LambdaFactories for creating an appropriate closure; a list of functions with their bodies in Go, a list of external services, etc.

The first operand almost always contains the location where we store the result of the operation.

Conditional operations are always of the form: "If <condition>, continue to the next operation; otherwise jump to <address>."

## The compiler

### Emitting bytecode

The compiler, as you would expect, goes along the ASTs generated by the parser of the function bodies etc, and turns them into bytecode. As it adds to the list of bytecode addresses, so it also adds to the list of memory locations, assigning one location to be a constant, another to hold the value of a variable, a third to hold the result of adding them together, etc.

As it goes from node to node, it passes itself a `Context` object. E.g. the top `CompileNode` method has a signature like this:

`func (cp *Compiler) CompileNode(node ast.Node, ctxt Context) cpResult`

The `Context` object keeps track of the bigger picture of what the compiler is trying to do by compiling the node: is it compiling a line it got from the REPL? The body of a command? The body of a function? Could this node be the return value of the function? What variables are in scope? Are we inside a `for` loop? Etc. It is therefore copied and modified from time to time as the compiler goes from node to node.

As you can see from the type signature above, compiling a node returns a `cpResult`. This contains information about the result type or types from compiling the node (about which more later); whether the result is foldable (i.e. whether it's a constant, modulo some special cases) and whether the compilation succeeded or failed.

The main `CompileNode` function of the compiler is just a big switch statement on the type of the node. (We are not at home to the Visitor Pattern.) However, instead of returning a `cpResult` as soon as its compiled the node, it stores the result in a `result` variable, so that after the end of the switch statement the compiler can do some sanity-checking and constant folding.

In compiling a block of code (the body of a function, or any sub-block) the compiler emits bytecode such that the result of the computation so far is in the last memory location in the list so far. This allows us to concatenate operations together without the second operation knowing what the first operation was; just as in a stack VM we arrange things so that the result is always on top of the stack.

Once a function has been compiled, information about how to call it is put into a `CpFunc` object, which is put in a list. Its index in this list is then put into the `Number` field of the `CallInfo` struct on the relevant leaf node of the correct `FunctionTree` in the compiler's `FunctionForest`. Now the compiler can find its way from a parsed function call to the address of the function in the VM and the locations of its operands.

### Backtracking and flow-of-control

The compiler, then, plows forward using up addresses and locations. But what happens when it needs to refer forward to addresses and locations as yet unassigned? Suppose for example we wish to compile something of the form:
```
condition :
    <code>
else :
    <other code>
```
In the bytecode, the `<code>` block needs to end with instructions to put its result into the location that will be on top of the memory after `<other code>` has finished compiling, and to jump to the address just after the top of the bytecode that will have been emitted. And since we don't know how much bytecode and memory will be taken up by compiling `<other code>`, we need to emit operations with dummy locations in, make a note of what we did, and then resolve it after compiling `<other code>`.

To do this, we have a range of functions for emitting various kinds of conditional code and returning a record of what was done, e.g. in implementing the logic for short-circuiting `or`, we create a `BkEarlyReturn` object like this:
```
shortCircuit := cp.VmConditionalEarlyReturn(vm.Qtru, leftRg, leftRg)
```
... which we later discharge by calling `cp.vmComeFrom(shortCircuit)`.

In hindsight, perhaps I should have written an intermediate representation for my code which mostly lowered it into bytecode, but retained some simple flow-of-control primitives. At this point it would take a lot of refactoring for little gain to revisit this, since the way we do flow-of-control is and should remain stable.

### Typechecking and constant folding

It isn't possible to completely typecheck any language dynamic enough that a function can conditionally return type A or type B. This follows from Rice's Theorem. You must either have false positives, where things are forbidden that would be safe at runtime --- in which case you have, in effect, a static type system.

*Or* you can have false negatives, where we throw an error at compile time only if the runtime *must* throw a type error.

This is what Pipefish does, and to do it effectively, when it compiles an expression, the compiler returns a `cpResult` struct (as discussed in the previous section) with a field `altType` containing a *range* of types that the compiled code might return on execution, stored in an `AlternateType`, which satisfies the `Typescheme` interface and which may itself contain a number of `Typeschemes`. These can contain information such as "either two strings, or an error" or "a tuple consisting of any number of integers", etc. This is way more fiddly than many typecheckers, but has proved surprisingly stable.

The result of all this is that when we consider compiling a function call `foo x, y`, and we have a function `foo(a int, y string)`, we can ask, *is it possible* that `x` is an `int` and `y` is a `string`, and then throw an error if it isn't.

The `cResult` struct also contains a field `foldable` whether the expression is foldable (as will usually be the case if all its operands are constant). We can then immediately do constant folding by running the expression it just compiled, rolling back the generated code and the memory addresses used, and put the result on top of memory as a constant.

### Multiple dispatch

This is handled in the `function_call.go file` of the `compiler` module.

In brief, what we do is, for each function call, having compiled the operands and worked out the types they could be, we use that information to navigate the non-backtracking `FunctionTree` we created in the intialization stage, finding out what if any logic we have to lower into the bytecode to do dispatch at runtime.

E.g. if we consider the same tree we looked at earlier:
```
type x == int :
	type y == int :
		"foo 2"
	type y == rune :
		"foo 1"
	else :
		error "no such implementation of foo"
else :
	type y == rune :
		"foo 1"
	else :
		error "no such implementation of foo"
```
... then if we happen to know at compile time that the first argument is an int and that the second isn't, but that it *might* be a rune, then the only logic we need at runtime can be produced by pruning the tree:
```
type y == rune :
		"foo 1"
	else :
		error "no such implementation of foo"
```
It is at the leaf nodes of the tree, in the `seekFunctionCall` function, that we differentiate between functions with their bodies in Pipefish compiled to bytecode, where the compiler needs to emit a `Call` operation saying where the bytecode is and where the arguments are and what to do with the result; and on the other hand functions which are builtins or constructors or calls to a Go function or an external service, which don't need a function call but just the emission of one or two bytecode operations.

By amalgamating the range of types we might get from each of the possible branches, we calculate the `AlternateType` the compiler should return; and it also returns the information that it's constant if all the operands were constant, so we can do constant folding.