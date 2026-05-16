## Preamble

### Contents of the file

This file describes the Pipefish VMs operations and their operands. It is the source of 
truth for this data in that the VM will also use this file to determine how many 
operands each operator should have and how to describe them when we call DescribeCode. It
will also used to add comments to the vm.go file.

The format of the file is that it contains this preabmble, ending with a line consisting of 
the subheading `## Operators`, and then a list of items in the following format.

(1) An empty line to mark the start of the item.
(2) A line consisting of the name of the operator, a separating colon and the flavors of 
    its operands.
(3) A short plain-English description of what it does.
(4) As many lines as we please of additional explanation.

E.g:

```
cpnt : dst mem
Codepoint of rune
Converts a rune into its Unicode code point, represented as an integer.
```

### Mnemonics for the operators

The four-letter mnemonics for the operators follow some loose conventions. Anything beginning with `q`
tests for a condition, and jumps to a given location if the condition is *not* met, otherwise it 
continues execution.

The last letter or sometimes two letters of the operator may indicate what sort of types the operator
works on: `i` for an integer, `f` for a float, `s` for a string, `S` for a set, `Sn` for a snippet,
`Z` for a struct, `L` for a list, `r` for a rune, `p` for a pair, `1` for a non-tuple, `x` for a type
that couldn't be determined at compile time, and `n` for an operand considered as a natural number.

### Mnemonics for the operand flavors

All operands are of type uint32, but they have different meanings depending on the operator: a meaning
depending essentially on which vector in the VM it is used to index. These are the "flavors" of operands, recorded in the list of operations below using the following three-letter mnemonics. Many of them are 
only used once: the common ones are dst, loc, mem, num, tok, tup, and typ.

* chk : The index number of a validation error data item in the vm's `ValidationErrors` vector.
* dst : The destination; an index to the VM's `Mem` vector saying where we put the result of an 
        operation.
* gfn : The index number of a Go function in the VM's `GoFn` vector.
* lfc : The index number of a lambda factory in the VM's `LambdaFactories` vector.
* loc : The index number of a location in the VM's `Code` vector.
* mem : The index number of an address in the VM's `Mem` vector, saying where one of the values
        the operation is working on is going to come from.
* num : The operand considered as a natural number rather than as an index to something in the VM.
* ptp : The index number of a map in the VM's `ParameterizedTypeInfo` vector.
* sfc : The index number of a snippet factory in the VM's `SnippetFactories` vector.
* tok : The index number of a token in the VM's `Tokens` vector.
* trk : The index number of static tracking info in the VM's `Tracking` vector.
* tup : Any number of uint32s, the meaning depending on context. This means that one `tup` operand
        may correspond to any number of actual arguments in the VM's `Run` method.
* typ : The index number of a type in the VM's `ConcreteTypeInfo` field.

### Conventions for explaining the operations

In the lines of additional explanation for each operation, the operands will be referred 
to as #0, #1, #2 etc. 

n#i will mean the value of #i, the 32-bit integer it contains.

m#i will mean the memory address indexed by operand #i.

v#i will mean the value stored in m#i.

E.g:

```
divi : dst mem mem tok
Divide ints 
Divides two integers and returns an integer, i.e. it implements `m div n`.
It returns an error constructed from token number n#3 if v#2 is 0. 
```

The items will be in alphabetical order of their operators.

## Operators

addf : dst mem mem
Add floats
Adds two floats, returning a float.

addi : dst mem mem
Add ints
Adds two ints, returning an int.

addL : dst mem mem
Add lists
Adds two lists, returning a list.

addS : dst mem mem
Add sets
Adds two sets, returning a set.

adds : dst mem mem
Add strings
Adds two floats, returning a float.

adrs : dst mem mem
Prepend rune to string

adsr : dst mem mem
Append rune to string

adtk : dst mem tok
Add token 
Adds a token to the trace of the error in mem

andb : dst mem mem
Boolean and

aref : dst mem
Assign to ref variable
Assigns v#1 to the reference variable in m#0.

asgm : dst mem
Assign to memory
Assigns v#1 to m#0.

auto : trk
Autogenerate tracking
Use tracking info number n#0 to generate tracking.

call : loc mem mem tup 
Function call 
Operands are:
    #0: the location to call.
    m#1 and m#2: the bottom and (exclusive) top of where to put the function's arguments.
    #3 a tuple of memory locations containing the values to put in the arguments.

calt : loc mem mem tup
Function call with tuple capture
This is like `call`, above, only with the possibility that it might be capturing a tuple, 
either by collecting up varargs or preventing a tuple from autosplatting.

casP : dst tok mem mem
Cast to parameterized clone type
Casts the value v#3 to the type v#2, where v#2 is a parameterized clone type.
Token n#1 can be used to return an error if the conversion is impossible.

cast : dst mem typ
Cast type
Casts v#1 to type number n#2.

casx : dst mem typ tok
Try to cast type
Like `cast`, except we don't know for certain it will succeed, so we also supply the
number of a token to throw an error if it can't be done.

cc11 : dst mem mem
Concatenate non-tuples

cc1T : dst mem mem
Concatenate non-tuple and tuple

ccT1 : dst mem mem
Concatenate tuple and non-tuple

ccTT : dst mem mem
Concatenate tuples

ccxx : dst mem mem
Concatenate unknowns
That is, either #1 or #2 may be a tuple or non-tuple, and we don't know which
at compile time.

chck : dst mem mem chk
Finish type validation
Operands are:
    v#0 : the value to be validated
    v#1 : evaluation of the validation condition, presumptively boolean
    v#2 : an int which is the number of the token of the calling constructor
	n#3 : the number of the validation error data
All this does is if v#1 is false, it constructs an error out of token number v#2 and
error number n#3, and overwrites the contents of m#0 with the error; otherwise it leaves
m#0 untouched.

chrf : dst mem
Check reference variable
At the end of executing a command, if it has reference variables, if we have inserted an
error into any of the reference variables, we must return the first of these errors instead
of `OK`. m#0 is the return location of the command; m#1 contains the reference variable.

clon : dst mem
Clones of type 
Implements `clones{T}`.

conL : dst mem mem
Append element to list 
Appends an element to a list, i.e. implements `L & x` where `L` is a list.

conS : dst mem mem
Add element to set
Adds an element to a list, i.e. implements `S & x` where `L` is a set.

cpnt : dst mem
Codepoint of rune
Converts a rune into its Unicode code point, represented as an integer.

cv1T : dst mem
Convert element to tuple
Converts v#1 to the tuple containing v#1.

cvTT : dst tup
Create tuple
Takes v#2 ... v#n and returns a tuple consisting of those elements.

diif : dst mem mem tok
Divide ints as float 
Divides two integers as a float, i.e. it implements `m / n` where `m` and `n` are 
integers. It returns an error constructed from token number n#3 if v#2 is 0. 

divf : dst mem mem tok
Divide floats 
Divides two floats, returning a float.
It returns an error constructed from token number n#3 if v#2 is 0.0. 

divi : dst mem mem tok
Divide ints 
Divides two integers and returns an integer, i.e. it implements `m div n`.
It returns an error constructed from token number n#3 if v#2 is 0. 

dvfi : dst mem mem tok
Divide float by int
Divides a float by an int and returns a float.
It returns an error constructed from token number n#3 if v#2 is 0. 

dvif : dst mem mem tok
Divide int by float
Divides an int by a float and returns a float.
It returns an error constructed from token number n#3 if v#2 is 0.0.

dofn : dst mem tup
Apply lambda function
Applies the function v#1 to the values in the tuple.

dref : dst mem
Dereference ref variable
Puts the contents of the reference variable in m#1 into m#0.

equb : dst mem mem
Boolean comparison with ==
Tests if two booleans are equal.

equf : dst mem mem
Float comparison with ==
Tests if two floats are equal.

equi : dst mem mem
Integer comparison with ==
Tests if two ints are equal.

equs : dst mem mem
String comparison with ==
Tests if two strings are equal

equt : dst mem mem
Type comparison with ==
Tests if two types are equal

eqxx : dst mem mem tok
Comparison with ==
Tests if two values are equal. If they are not comparable, we return an error
based on token number n#3.

eval : dst mem num
Eval
This evaluates the string v#1 using evaluator number n#2.

extn : dst num num mem mem tup
External service call
Operands are: 
    n#1 : the number of the external service to call
    n#2 : whether the function being called is a prefix, infix, postfix or unfix.
    v#2 : the remainder of the namespace of the function as a string
    v#3 : the name of the function as a string
    #4 : a tuple of the locations of the arguments we wish to pass.

flpp :
Pop peek flags

flps : mem
Push peek flags
v#0 will be of internal type PEEK_FLAGS.

flti : dst mem
Float from int

flts : dst mem tok
Float from string
Token number n#2 is used to make an error if the conversion fails.

gofn : dst mem gfn tup
Call Go function
Operands are :
    m#1 : contains an error which we will doctor before (if necessary) returning it.
    n#2 : the number of the Go function we want to call.
    #3 : a tuple of the locations of the arguments we want to pass to the function.

gsql : dst mem mem mem mem num tok
Get from SQL
This returns an error or `OK` in m#0, the SQL data being put in the reference variable v#1.
Operands are :
    v#1 : the address of the reference variable: where we put what we get from SQL.
    v#2 : the desired type of the result
	v#3 : the database connection
	v#4 : the snippet of SQL
	n#5 : 0 for `get as`, 1 for `get like`.
	n#6 : the number of a token for emitting an error if required.

gtef : dst mem mem
Float comparison with >=

gtei : dst mem mem
Int comparison with >=

gthf : dst mem mem
Float comparison with >

gthi : dst mem mem
Int comparison with >

idxL : dst mem mem tok
Index list 
v#1 is the list, v#2 is an integer, and n#3 is the number of a token to make an error
in the case that v#2 is out of bounds.

idxp : dst mem mem tok
Index pair 
v#1 is the pair, v#2 is an integer, and n#3 is the number of a token to make an error
in the case that v#2 is out of bounds.

idxs : dst mem mem tok
Index string
v#1 is the string, v#2 is an integer, and n#3 is the number of a token to make an error
in the case that v#2 is out of bounds.

idxT : dst mem mem tok
Index tuple
v#1 is the tuple, v#2 is an integer, and n#3 is the number of a token to make an error
in the case that v#2 is out of bounds.

ixSn : dst mem mem tok
Index snippet
v#1 is the tuple, v#2 is an integer, and n#3 is the number of a token to make an error
in the case that v#2 is out of bounds.

ixTn : dst mem num
Hard-index tuple
v#1 is the tuple, and we index it by n#2.

ixZl : dst mem mem tok
Index struct by label
v#1 is the struct, v#2 is a label, and n#3 is the number of a token to make an error
in the case that v#1 has no field labeled by v#2.

ixZn : dst mem num
Hard-index struct
v#1 is the struct and n#2 is the number of the field we want to index, determined at
compile-time.

inpt : dst mem mem
Input from keyboard
v#1 is of type `terminal.Keyboard` with one field consisting of the prompt. #v2 is a 
boolean saying whether the input should be masked for privacy.

inxL : dst mem mem
Is element in list

inxS : dst mem mem
Is element in set

inxt : dst mem mem
Is element in type

inxT : dst mem mem
Is element in tuple

inte : dst mem
Integer from enum

intf : dst mem
Integer from float

ints : dst mem tok
Integer from string 
n#2 is the number of a token to make an error if conversion fails.

itgk : dst mem
Get key from iterator

itgv : dst mem
Get value from iterator

itkv : dst dst mem
Get key and value from iterator

itor : dst mem
Integer to rune

inxS : dst mem num
Hard-index snippet
Returns element number n#2 of snippet v#1.

ixXx : dst mem mem tok
Index value by value
In the case where at compile time we can't determine the types of one or other or both
of v#1 and v#2. The token number n#3 can be used to create an error if the index is the 
wrong type or out of bounds.

jmp : loc
Jump

json : dst mem mem num tok
Json to Pipefish
Operands are :
    v#1 : a string containing the JSON.
    v#2 : the type to convert to.
    n#3 : 0 or 1 to indicate whether we are converting "like" or "as".
    n#4 : the number of a token for constructing an error in the case the conversion fails.

jsr : loc
Jump to subroutine
Pushes the location we're jumping from onto the stack, so that `rtn` will return to just after
the jump.

keyM : dst mem
Keys of map
Returned as a list.

keyZ : dst mem
Keys of struct
Returned as a list containing the labels.

lbls : dst mem tok
Label from string
Returns token number n#2 if the conversion fails.

lenL : dst mem
Length of list

lenM : dst mem
Length of map

lens : dst mem
Length of string

lenS : dst mem
Length of set

lenT : dst mem
Length of tuple

list : dst mem
List from tuple

litx : dst mem num tok
Literal of value
Operands :
    v#1 is the value.
    n#2 is the number of the compiler to generate the literal.
    n#3 is the number of a token for error-generation if the value has no literal representation.

lnSn : dst mem
Length of snippet

logn : 
Turn logging off

logy : 
Turn logging on

mkEn : dst typ mem tok
Enum element from int
Makes an enum of type number n#1 from an integer v#2, using token n#3 to return an error if
v#2 is out of bounds.

mker : dst mem tok
Error from string

mkfn : dst lfc
Make lambda
Here n#1 is the number of a lambda factory which knows how to make the lambda.

mkit : dst mem num tok
Make iterator
v#1 is the range of the iterator, n#2 is 0 or 1 according to whether the iterator  doesn't or
does only return keys, and token number n#3 is used to create a runtime error if the range
is invalid.

mkmp : dst mem tok
Make map
Constructs a map from a tuple value v#1, using token n#2 to create an error if the
elements of the tuple have the wrong type, e.g. there's an unhashable key.

mkpr : dst mem mem
Make pair

mkSn : dst sfc
Make snippet
Here n#1 is a snippet factory analogous to a lambda factory.

mkst : dst mem tok
Make set
Constructs a map from a tuple value v#1, using token n#2 to create an error if the
elements of the tuple have the wrong type, e.g. there's an unhashable value.

modi : dst mem mem tok
Modulus of integers

mpar : dst ptp tok tup
Make parameterized type
Operands :
    n#1 : the number of the parameterized type constructor.
    n#2 : number of a token for throwing an error if the value can't be constructed.
    n#3 : a tuple of arguments to pass to the constructor.

mulf : dst mem mem
Multiply floats

muli : dst mem mem
Multiply ints

negf : dst mem
Negate flaot

negi : dst mem
Negate int

notb : dst mem
Binary not

outp : mem
Post to output

outt : mem
Post to terminal

psql : dst mem mem tok
Post to SQL
Here v#1 is the SQL accessor object and v#2 is a snippet. We return either `OK` or an 
error created using the token n#3

qabt : mem tup loc
Test abstract type 
Jumps to the location n#2 if the type of v#0 is not in the tuple of type numbers in #1.

qfls : mem loc
Test for false
This jumps to the location number n#1 if v#0 is not false.

qitr : mem loc
Test for end of iterator
This jumps to location number n#1 if the iterator v#0 hasn't finished iterating.

qleT : mem num loc
Test length of tuple <= n
Jumps to location number n#2 if the length of the tuple value v#0 isn't less than or equal to n#1.

qlnT : mem num loc
Test length of tuple < n
Jumps to location number n#2 if the length of the tuple value v#0 isn't less than or equal to n#1.

qlog : loc
Jumps to location number n#0 if logging is turned off

qnab : mem tup loc
Test not in abstract type 
Jumps to the location n#2 if the type of v#0 is in the tuple of type numbers in #1.

qntp : mem typ loc
Test not of type 
Jumps to the location n#2 if the type of v#0 is not the type numbers in n#1.

qsat : mem loc
Test satisfied
Jumps to location n#1 if v#0 is an unsatisfied conditional.

qsnq : mem loc
Test singleton
Jumps to location n#1 if v#2 is a tuple.

qtpt : mem num tup loc
Test tuple types
Jumps to location n#3 if the first n#1 elements of the tuple value v#0 don't have types corresponding
to the type numbers in n#2.

qtru : mem loc
Test true
Jumps to location n#1 if v#0 isn't true

qtyp : mem typ loc
Test type membership
Jumps to location #2 if v#0 doesn't have type number n#1

ret : 
Return
If the height of the return stack is strictly greater than it was when the vm's `.Run` 
method was called, then we pop the top off the return stack and jump to that location.
Otherwise we've finished exacuting `.Run` and can return.

rpop : 
Pop recursion data

rpsh : num num
Push recursion data

sliL : dst mem mem tok
Slice of list
v#1 is a list, v#2 is a pair; token n#3 is used to create errors for e.g. when the pair
is out of bounds.

slis : dst mem mem tok
Slice of string
v#1 is a string, v#2 is a pair; token n#3 is used to create errors for e.g. when the pair
is out of bounds.

sliT : dst mem mem tok
Slice of tuple
v#1 is a tuple, v#2 is a pair; token n#3 is used to create errors for e.g. when the pair
is out of bounds.

slTn : dst mem num
Hard slice tuple
Returns the tuple consisting of the elements from n#1 to the end of the tuple value v#1.

strc : dst typ tup
Make struct
Constructs a struct of type n#1 from the values in the memory locations given in #2.

strP : dst tok tup
Make parameterized struct.
Constructs a parameterized struct of type n#1 from the values in the memory locations given in #2.
An error will be constructed from token number n#1 if the struct can't be constructed.

strx : dst mem
String of value

subf : dst mem mem
Subtract floats

subi : dst mem mem
Subtract integers

subS : dst mem mem
Subtract sets

thnk : dst mem loc
Initialize thunk
This will set m#0 to be a value of type THUNK with a payload of n#1 and n#2. The first of these
says where the result of unthinking the thunk will end up; the second says where the VM will have 
to jsr to to unthunk it.

tinf : dst mem
Get info for type
This dumps the type info for the type into a list returned in m#0. This is done under the hood,
the user never sees the raw list.

tupf : dst mem tok
First element of tuple
Returns the first element of a tuple, or an element created from token n#2 if the tuple is empty.

trak : trk
Make tracking data
This constructs live tracking data saying what the compiler is doing now from the static tracking 
datum number n#0

tupL : dst mem
Tuple of list
That is, this implements the splat operator `L ...`.

tuLx : dst mem tok
Tuple of possible list
Splats the list if it is a list, otherwise returns an error constructed from token t#2.

typu : dst mem mem
Type union
That is, this implements `typeA/typeB`.

typx : dst mem
Type of value

untE : dst mem
Unthunk error
This takes the error v#1, converts all the arguments of the error of type uint32 to the values
in the corresponding memory locations, and returns it in m#0.

untk : dst
Unthunk
This checks whether v#1 is of type THUNK. If it is, it `jsr`s to the code address contained in the thunk,
gets the evaluated result of the thunk, and puts it into m#0; otherwise it does nothing.

uwrp : dst mem tok
Unwrap error
This turns something of type `error` into something of type `Error`, and ordinary struct defined in the
builtins. The token n#3 is used to return an error if we're trying to unwrap something that is not in
fact of type error

vlid : dst mem
Valid
Returns `true` if v#1 is not of type error

wrHb : mem mem
Write to hub
A magical gizmo that lets services which are also hubs tell hub.go what to do. v#0 is a string saying
which hub action we want to take, and v#2 is a list containing parameters.

wthL : dst mem tok tup
List with
The `with` operator for lists. v#1 is a list, #2 is a tuple of pairs, and token n#2 is for constructing
an error if the pairs are wrong, e.g. if the key of a pair is outside the bounds of the list.

wthM : dst mem tok tup
Map with
The `with` operator for maps. v#1 is a map, #2 is a tuple of pairs, and token n#2 is for constructing
an error if the pairs are wrong, e.g. if the key of a pair is unhashable.

wthT : dst mem tok tup
Tuple with
The `with` operator for tuples. v#1 is a tuple, #2 is a tuple of pairs, and token n#2 is for constructing
an error if the pairs are wrong, e.g. if the key of a pair is outside the bounds of the tuple.

wthZ : dst mem tok tup
Struct with
The `with` operator for structs. v#1 is a struct, #2 is a tuple of pairs, and token n#2 is for constructing
an error if the pairs are wrong, e.g. if the key of a pair is not a field of the struct.

wtoM : dst mem tok tup
Map without
v#1 is a map, and #3 is a tuple of key values to be removed from it. Token n#2 is for constructing
an error if any of the values is unhashable unhashable

yeet : dst mem
Yeet type parameters
If v#1 is of a parameterized type, then this assigns the parameters of this type to m#0 and to the
following memory addresses, one address for each parameter.