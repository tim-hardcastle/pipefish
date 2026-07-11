package vm

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"

	"src.elv.sh/pkg/persistent/vector"

	"github.com/tim-hardcastle/pipefish/source/err"
	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/text"
	"github.com/tim-hardcastle/pipefish/source/token"
	"github.com/tim-hardcastle/pipefish/source/values"
)

type Vm struct {
	// Temporary state: things we change at runtime.
	Mem            []values.Value
	Code           []*Operation
	callstack      []uint32
	recursionStack []recursionData
	logging        bool
	// TODO --- the LogToLoc field of TrackingData is never used by *live* tracking, which should therefore have its own data type.
	LiveTracking []TrackingData // "Live" tracking data in which the uint32s in the permanent tracking data have been replaced by the corresponding memory registers.
	PostHappened bool

	// Permanent state: things established at compile time.

	// These are things the ordinal of which can be an operand.
	Tokens           []*token.Token
	LambdaFactories  []*LambdaFactory
	SnippetFactories []*SnippetFactory
	GoFns            []GoFn
	Evaluators       []func(string) values.Value // One per compiler, to implement `eval`.

	// As sometimes can this; it is indexed by the numbers of concrete types.
	ConcreteTypeInfo []TypeInformation

	// This contains the information necessary to attach a suitable namespace to the literal
	// of a value; it is indexed first by the number of the compiler and second by the number of
	// the type.
	NamespaceInfo []map[values.ValueType]string
	// This contains the information necessary to call the tests of a given compiler.
	Tests         [][]TestInfo

	Labels                     []string // Array from the number of a field label to its name.
	ValidationErrors           []*ValidationError
	Tracking                   []TrackingData // Data needed by the 'trak' opcode to produce the live tracking data.
	InHandle                   InHandler
	OutHandle                  OutHandler
	AbstractTypes              []AbstractTypeInfo
	ExternalCallHandlers       []ExternalCallHandler // The services declared external, whether on the same hub or a different one.
	UsefulTypes                UsefulTypes
	UsefulValues               UsefulValues
	TypeNumberOfUnwrappedError values.ValueType  // What it says. When we unwrap an 'error' to an 'Error' struct, the vm needs to know the number of the struct.
	StringifyLoReg             uint32            // |
	StringifyCallTo            uint32            // | These are so the vm knows how to call the stringify function.
	StringifyOutReg            uint32            // |
	FieldLabelsInMem           map[string]uint32 // Used to turn a string into a label.
	ParameterizedTypeInfo      []values.Map      // A list of maps from type parameters (as TUPLE values) to types (as TYPE values). The list is itself keyed by a map from type operators to the position in the list, which is stored in the compiler.

	// Things for converting to and from Go.
	GoToPipefishTypes map[reflect.Type]values.ValueType
	GoConverter       [](func(t uint32, v any) any)
	GoEquals          func(x any, y any) bool
	GoLiteral         func(x any) string

	// Controls dumping the compiler and VM.
	//
	PeekStack   []map[string]bool // Flags for peeking the compiler and VM.
	OutputTo    string            // Gives a filename to dump output to.
	IndentBy    int               // Indentation to allow us to display the children of a node att a different depth.
	IsCompiling bool              // So we can optionally only dump the VM during compilation, i.e. when it's doing constant folding.
}

// In general, the VM can't convert from type names to type numbers, because it doesn't
// need to. And we don't need the whole map of them because only a tiny proportion are
// needed by the runtime, so a struct gives us quick access to what we do need.
type UsefulTypes struct {
	UnwrappedError values.ValueType
	LogTo          values.ValueType
}

// Similarly we need to know where some values are kept, if they have special effects
// on runtime behavior.
type UsefulValues struct {
	OutputAs uint32
}

type TestInfo struct {
	CallTo uint32  // The address to call to run a given test.
	Return uint32  // Where it puts its return value.
}

// Contains a Go function in the form of a reflect.Value, and, currently, nothing else.
// TODO --- this has been the case for a long time, you could probably refactor now.
type GoFn struct {
	Code reflect.Value
}

type AbstractTypeInfo struct {
	Name string
	Path string
	AT   values.AbstractType
	IsMI bool
}

func (aT AbstractTypeInfo) IsMandatoryImport() bool {
	return aT.IsMI
}

// Contains the information to execute a lambda at runtime; i.e. it is the payload of a FUNC type value.
type Lambda struct {
	CapturesStart  uint32
	CapturesEnd    uint32
	ParametersEnd  uint32
	ResultLocation uint32
	AddressToCall  uint32
	Captures       []values.Value
	Sig            []values.AbstractType // To represent the call signature. Unusual in that the types of the AbstractType will be nil in case the type is 'any?'
	RtnSig         []values.AbstractType // The return signature. If empty means ok/error for a command, anything for a function.
	Tok            *token.Token
	Gocode         *reflect.Value // If it's a lambda returned from Go code, this will be non-nil, and most of the other fields will be their zero value except the sig information.
}

// Interface wrapping around external calls whether to the same hub or via HTTP.
type ExternalCallHandler interface {
	Evaluate(line string) values.Value
	Problem() *err.Error
	GetAPI() string
}

// All the information we need to make a lambda at a particular point in the code.
type LambdaFactory struct {
	Model            *Lambda  // Copy this to make the lambda.
	CaptureLocations []uint32 // Then these are the location of the values we're closing over, so we copy them into the lambda.
}

// All the information we need to make a snippet at a particular point in the code.
// Currently contains only the bindle but later may contain some secret sauce.
type SnippetFactory struct {
	Bindle *values.SnippetBindle // Points to the structure defined below.
}

// For containing the data needed to manufacture a typechecking error at runtime.
type ValidationError struct {
	Tok       *token.Token
	Condition string
	Type      string
	Value     uint32
}

// Container for the data we push when a function might be about to do recursion.
type recursionData struct {
	mems []values.Value
	loc  uint32
}

// Used for injecting data into HTML.
type HTMLInjector struct {
	Data []any
}

// This is used for unthinking errors, since the `err` package can't describe the type of a value.
type DescribeTypeOfValueAtLocation uint32

// These inhabit the first few memory addresses of the VM.
var CONSTANTS = []values.Value{values.UNDEF, values.FALSE, values.TRUE, values.U_OBJ, values.ZERO, values.ONE, values.BLNG, values.OK, values.EMPTY}

// Type names in upper case are things the user should never see.
var nativeTypeNames = []string{"UNDEFINED VALUE", "INT ARRAY", "THUNK", "CREATED LOCAL CONSTANT",
	"BLING", "UNSATISFIED CONDITIONAL", "REFERENCE VARIABLE",
	"ITERATOR", "PEEK", "ok", "tuple", "error", "null", "int", "bool", "string", "rune", "float", "type", "func",
	"pair", "list", "map", "set", "label", "snippet"}

func BlankVm() *Vm {
	vm := &Vm{Mem: make([]values.Value, len(CONSTANTS)),
		logging:           true,
		InHandle:          &StandardInHandler{"→ ", nil},
		GoToPipefishTypes: map[reflect.Type]values.ValueType{},
		GoConverter:       [](func(t uint32, v any) any){},
		NamespaceInfo:     []map[values.ValueType]string{},
		FieldLabelsInMem:  make(map[string]uint32),
		PeekStack:         []map[string]bool{},
	}
	vm.OutHandle = &SimpleOutHandler{os.Stdout, vm}
	copy(vm.Mem, CONSTANTS)
	for _, name := range nativeTypeNames {
		vm.ConcreteTypeInfo = append(vm.ConcreteTypeInfo, BuiltinType{name: name})
	}
	vm.UsefulTypes.UnwrappedError = DUMMY
	vm.Mem = append(vm.Mem, values.Value{values.SUCCESSFUL_VALUE, nil}) // TODO --- why?
	return vm
}

// The `run` function can and occasionally does call itself. If we want to catch a panic
// from the VM, we need to unwind the stack back to where we actually entered. This is what
// `Run` is for: everything calling `run` does so through `Run`, except `run` itself and
// a few other places in the VM itself.
//
// In the same way we catch Ctrl+C interupts and return an appropriate error.
func (vm *Vm) Run(loc uint32) {
	// First the panics.
	if !settings.ALLOW_PANICS {
		defer func() {
			if r := recover(); r != nil {
				e := err.CreateErr("vm/panic", &token.Token{}, fmt.Sprintf("%v", r))
				vm.Mem = append(vm.Mem, values.Value{values.ERROR, e})
			}
		}()
	}
	// Then the ctrl+c interrupts.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		e := err.CreateErr("vm/ctrl/c", &token.Token{})
		vm.Mem = append(vm.Mem, values.Value{values.ERROR, e})
	}()
	vm.run(loc, ctx, c)
}

// The heart of the VM. A big loop around a switch. It will keep going until it hits a `ret`
// and the callstack is the same height as when it was called.
//
// (This condition, rather than just saying "until the callstack is empty" allows `run` to
// call itself under certain rare and harmless conditions.)
//
// The comments to the right of and immediately below each `case` statement in the main
// `switch` statement are auto-generated from the documentation in `operations.md` and should
// be edited there and not here. Other comments are safe to edit.
//
// For the meanings of the operand "flavors", `dst`, `mem`, etc, see `operations.md`.
func (vm *Vm) run(loc uint32, ctx context.Context, cancel chan os.Signal) {
	// We exit the loop and this function when we perform a `ret` openeration and `stackHeight``
	// equals the length of the callstack.
	stackHeight := len(vm.callstack)
loop:
	for {
		select {
		case <-ctx.Done():
			vm.callstack = vm.callstack[0:stackHeight]
			return
		default:
			// We do this now and by hand so as to avoid commenting Flpp when possible.
			if settings.PEEK_VM && vm.Code[loc].Opcode == Flpp {
				vm.PopPeeks()
				loc++
				continue loop
			}
			if settings.PEEK_VM && vm.IsSet("c") || (vm.IsSet("k") && vm.IsCompiling) {
				vm.Dump("! " + vm.DescribeCode(loc))
			}
			if (settings.PEEK_VM && vm.IsSet("c") || (vm.IsSet("k") && vm.IsCompiling)) && !vm.IsSet("s") {
				vm.Dump(vm.DescribeOperandValues(loc))
			}
			args := vm.Code[loc].Args
		Switch:
			switch vm.Code[loc].Opcode {
			case Addf: // Add floats (dst mem mem)
				// Adds two floats, returning a float.
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(float64) + vm.Mem[args[2]].V.(float64)}
			case Addi: // Add ints (dst mem mem)
				// Adds two ints, returning an int.
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(int) + vm.Mem[args[2]].V.(int)}
			case AddL: // Add lists (dst mem mem)
				// Adds two lists, returning a list.
				result := vm.Mem[args[1]].V.(vector.Vector)
				rhs := vm.Mem[args[2]].V.(vector.Vector)
				for i := 0; ; i++ {
					el, ok := rhs.Index(i)
					if !ok {
						break
					}
					result = result.Conj(el)
				}
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, result}
			case AddS: // Add sets (dst mem mem)
				// Adds two sets, returning a set.
				result := vm.Mem[args[1]].V.(values.Set).Union(vm.Mem[args[2]].V.(values.Set))
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, result}
			case Adds: // Add strings (dst mem mem)
				// Adds two floats, returning a float.
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(string) + vm.Mem[args[2]].V.(string)}
			case Adrs: // Prepend rune to string (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.STRING, string(vm.Mem[args[1]].V.(rune)) + vm.Mem[args[2]].V.(string)}
			case Adsr: // Append rune to string (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.STRING, vm.Mem[args[1]].V.(string) + string(vm.Mem[args[2]].V.(rune))}
			case Adtk: // Add token  (dst mem tok)
				// Adds a token to the trace of the error in mem
				vm.Mem[args[0]] = vm.Mem[args[1]]
				vm.Mem[args[0]].V.(*err.Error).AddToTrace(vm.Tokens[args[2]])
			case Andb: // Boolean and (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(bool) && vm.Mem[args[2]].V.(bool)}
			case Aref: // Assign to ref variable (dst mem)
				// Assigns v#1 to the reference variable in m#0.
				vm.Mem[vm.Mem[args[0]].V.(uint32)] = vm.Mem[args[1]]
			case Asgm: // Assign to memory (dst mem)
				// Assigns v#1 to m#0.
				vm.Mem[args[0]] = vm.Mem[args[1]]
			case Auto: // Autogenerate tracking (trk)
				// Use tracking info number n#0 to generate tracking.
				if vm.logging {
					staticData := vm.Tracking[args[0]]
					newData := TrackingData{staticData.Flavor, staticData.Tok, staticData.LogToLoc, staticData.LogTimeLoc, make([]any, len(staticData.Args))}
					copy(newData.Args, staticData.Args) // This is because only things of type uint32 are meant to be replaced.
					for i, v := range newData.Args {
						if v, ok := v.(uint32); ok {
							newData.Args[i] = vm.Mem[v]
						}
					}
					trackingString := vm.TrackingToString([]TrackingData{newData})
					switch vm.Mem[staticData.LogToLoc].T {
					case vm.UsefulTypes.LogTo:
						if vm.Mem[staticData.LogToLoc].V.(int) == 0 {
							println(text.NewMarkdown("", 92, func(s string) string { return s }).Render([]string{trackingString}))
						} else {
							vm.OutHandle.Write(text.NewMarkdown("", 92, func(s string) string { return s }).Render([]string{trackingString}))
						}
					case values.STRING:
						filename := vm.Mem[staticData.LogToLoc].V.(string)
						path := filename
						if filepath.IsLocal(path) {
							path = filepath.Join(filepath.Dir(staticData.Tok.Source), path)
						}
						var f *os.File
						if _, err := os.Stat(path); os.IsNotExist(err) {
							f, err = os.Create(path)
							if err != nil {
								panic(err)
							}
						} else {
							f, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0660)
							if err != nil {
								panic(err)
							}
						}
						f.WriteString(trackingString)
					}
				}
			case Call: // Function call  (loc mem mem tup)
				// Operands are:
				//     #0: the location to call.
				//     m#1 and m#2: the bottom and (exclusive) top of where to put the function's arguments.
				//     #3 a tuple of memory locations containing the values to put in the arguments.
				paramNumber := args[1]
				argNumber := 3
				for paramNumber < args[2] {
					v := vm.Mem[args[argNumber]]
					if v.T == values.TUPLE {
						tup := v.V.([]values.Value)
						for ix := 0; ix < len(tup); ix++ {
							vm.Mem[paramNumber] = tup[ix]
							paramNumber++
						}
						argNumber++
					} else {
						vm.Mem[paramNumber] = v
						paramNumber++
						argNumber++
					}
				}
				vm.callstack = append(vm.callstack, loc)
				loc = args[0]
				continue
			case CalT: // Function call with tuple capture (loc mem mem tup)
				// This is like `call`, above, only with the possibility that it might be capturing a tuple, 
				// either by collecting up varargs or preventing a tuple from autosplatting.
				paramNumber := args[1]
				argNumber := 3
				tupleOrVarargsData := vm.Mem[args[2]].V.([]uint32)
				var varargsTime bool
				for paramNumber < args[2] {
					torvIndex := paramNumber - args[1]
					if tupleOrVarargsData[torvIndex] == 1 && !varargsTime {
						vm.Mem[paramNumber] = values.Value{values.TUPLE, []values.Value{}}
						varargsTime = true
					}
					if varargsTime && len(args) <= argNumber { // Then we have no more arguments but may be supplying an empty varargs.
						paramNumber++
						continue
					}
					v := vm.Mem[args[argNumber]]
					if v.T == values.TUPLE && tupleOrVarargsData[torvIndex] != 2 { // Then we're exploding a tuple.
						tup := v.V.([]values.Value)
						if varargsTime { // We may be doing a varargs, in which case we suck the whole tuple up into the vararg.
							vararg := vm.Mem[paramNumber].V.([]values.Value)
							vm.Mem[paramNumber].V = append(vararg, tup...)
						} else { // Otherwise we need to explode it and put it into the parameters one at a time unless and untill we run out of them or we meet a varargs.
							for ix := 0; ix < len(tup); ix++ {
								if tupleOrVarargsData[paramNumber-args[1]] == 1 { // The vararg will slurp up what remains of the tuple.
									varargsTime = true
									vm.Mem[paramNumber] = values.Value{values.TUPLE, tup[ix:]}
									break
								}
								vm.Mem[paramNumber] = tup[ix]
								paramNumber++
							}
						}
						argNumber++
					} else { // Otherwise we're not exploding a tuple.
						if varargsTime {
							for (argNumber < len(args)) && vm.Mem[args[argNumber]].T != values.BLING {
								vararg := vm.Mem[paramNumber].V.([]values.Value)
								if vm.Mem[args[argNumber]].T == values.TUPLE {
									vm.Mem[paramNumber].V = append(vararg, vm.Mem[args[argNumber]].V.([]values.Value)...)
								} else {
									vm.Mem[paramNumber].V = append(vararg, vm.Mem[args[argNumber]])
								}
								argNumber++
							}
							varargsTime = false
							paramNumber++
						} else {
							vm.Mem[paramNumber] = v
							paramNumber++
							argNumber++
						}
					}
				}
				vm.callstack = append(vm.callstack, loc)
				loc = args[0]
				continue
			case CasP: // Cast to parameterized clone type (dst tok mem mem)
				// Casts the value v#3 to the type v#2, where v#2 is a parameterized clone type.
				// Token n#1 can be used to return an error if the conversion is impossible.
				abtype := vm.Mem[args[2]].V.(values.AbstractType).Types
				if len(abtype) != 0 {
					vm.Mem[args[0]] = vm.makeError("vm/cast/concrete.b", args[1])
				}
				typeNo := vm.Mem[args[2]].V.(values.AbstractType).Types[0]
				if typeCheck := vm.ConcreteTypeInfo[typeNo].(CloneType).Validation; typeCheck != nil {
					vm.Mem[typeCheck.TokNumberLoc] = values.Value{values.INT, int(args[1])}
					vm.Mem[typeCheck.InLoc] = vm.Mem[args[3]]
					vm.Mem[typeCheck.ResultLoc] = values.Value{typeNo, vm.Mem[args[3]].V}
					vm.run(typeCheck.CallAddress, ctx, cancel)
					vm.Mem[args[0]] = vm.Mem[typeCheck.ResultLoc]
				} else {
					vm.Mem[args[0]] = values.Value{typeNo, vm.Mem[args[3]].V}
				}
			case Cast: // Cast type (dst mem typ)
				// Casts v#1 to type number n#2.
				vm.Mem[args[0]] = values.Value{values.ValueType(args[2]), vm.Mem[args[1]].V}
			case Casx: // Try to cast type (dst mem typ tok)
				// Like `cast`, except we don't know for certain it will succeed, so we also supply the
				// number of a token to throw an error if it can't be done.
				currentType := vm.Mem[args[1]].T
				if currentType == values.ERROR {
					vm.Mem[args[0]] = vm.Mem[args[1]]
					break Switch
				}
				castToAbstract := vm.Mem[args[2]].V.(values.AbstractType)
				if len(castToAbstract.Types) != 1 {
					vm.Mem[args[0]] = vm.makeError("vm/cast/concrete", args[3], args[1], args[2])
					break Switch
				}
				targetType := castToAbstract.Types[0]
				if targetType == currentType {
					vm.Mem[args[0]] = vm.Mem[args[1]]
					break Switch
				}
				if enumInfo, ok := vm.ConcreteTypeInfo[targetType].(EnumType); ok && currentType == values.INT {
					if vm.Mem[args[1]].V.(int) >= len(enumInfo.ElementNames) || vm.Mem[args[1]].V.(int) < 0 {
						vm.Mem[args[0]] = vm.makeError("vm/cast/enum", args[3], args[1], args[2])
						break Switch
					}
					vm.Mem[args[0]] = values.Value{targetType, vm.Mem[args[1]].V.(int)}
					break Switch
				}
				if structInfo, ok := vm.ConcreteTypeInfo[targetType].(StructType); ok && currentType == values.LIST {
					elements := vm.Mem[args[1]].V.(vector.Vector)
					if elements.Len() != len(structInfo.AbstractStructFields) {
						vm.Mem[args[0]] = vm.makeError("vm/cast/fields", args[3], args[1], args[2])
						break Switch
					}
					fields := make([]values.Value, elements.Len())
					for i := 0; i < elements.Len(); i++ {
						el, _ := elements.Index(i)
						if !structInfo.AbstractStructFields[i].Contains(el.(values.Value).T) {
							vm.Mem[args[0]] = vm.makeError("vm/cast/types", args[3], args[1], args[2])
							break Switch
						}
						fields[i] = el.(values.Value)
					}
					vm.Mem[args[0]] = values.Value{targetType, fields}
					break Switch
				}
				if cloneInfoForCurrentType, ok := vm.ConcreteTypeInfo[currentType].(CloneType); ok {
					if cloneInfoForCurrentType.Parent == targetType {
						vm.Mem[args[0]] = values.Value{targetType, vm.Mem[args[1]].V}
						break Switch
					}
					if cloneInfoForTargetType, ok := vm.ConcreteTypeInfo[currentType].(CloneType); ok && cloneInfoForTargetType.Parent == cloneInfoForCurrentType.Parent {
						vm.Mem[args[0]] = values.Value{targetType, vm.Mem[args[1]].V}
						break Switch
					}
				}
				// Otherwise by elimination the current type is the parent and the target type is a clone, or we have an error.
				if cloneInfoForTargetType, ok := vm.ConcreteTypeInfo[targetType].(CloneType); ok && cloneInfoForTargetType.Parent == currentType {
					vm.Mem[args[0]] = values.Value{targetType, vm.Mem[args[1]].V}
					break Switch
				}
				vm.Mem[args[0]] = vm.makeError("vm/cast", args[3], args[1], args[2])
			case Cc11: // Concatenate non-tuples (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.TUPLE, []values.Value{vm.Mem[args[1]], vm.Mem[args[2]]}}
			case Cc1T: // Concatenate non-tuple and tuple (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.TUPLE, append([]values.Value{vm.Mem[args[1]]}, vm.Mem[args[2]].V.([]values.Value)...)}
			case CcT1: // Concatenate tuple and non-tuple (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.TUPLE, append(vm.Mem[args[1]].V.([]values.Value), vm.Mem[args[2]])}
			case CcTT: // Concatenate tuples (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.TUPLE, append(vm.Mem[args[1]].V.([]values.Value), vm.Mem[args[2]].V.([]values.Value)...)}
			case Ccxx: // Concatenate unknowns (dst mem mem)
				// That is, either #1 or #2 may be a tuple or non-tuple, and we don't know which
				// at compile time.
				if vm.Mem[args[1]].T == values.TUPLE {
					if vm.Mem[args[2]].T == values.TUPLE {
						vm.Mem[args[0]] = values.Value{values.TUPLE, append(vm.Mem[args[1]].V.([]values.Value), vm.Mem[args[2]].V.([]values.Value)...)}
					} else {
						vm.Mem[args[0]] = values.Value{values.TUPLE, append(vm.Mem[args[1]].V.([]values.Value), vm.Mem[args[2]])}
					}
				} else {
					if vm.Mem[args[2]].T == values.TUPLE {
						vm.Mem[args[0]] = values.Value{values.TUPLE, append([]values.Value{vm.Mem[args[1]]}, vm.Mem[args[2]].V.([]values.Value)...)}
					} else {
						vm.Mem[args[0]] = values.Value{values.TUPLE, []values.Value{vm.Mem[args[1]], vm.Mem[args[2]]}}
					}
				}
			case Chck: // Finish type validation (dst mem mem chk)
				// Operands are:
				//     v#0 : the value to be validated
				//     v#1 : evaluation of the validation condition, presumptively boolean
				//     v#2 : an int which is the number of the token of the calling constructor
				// 	n#3 : the number of the validation error data
				// All this does is if v#1 is false, it constructs an error out of token number v#2 and
				// error number n#3, and overwrites the contents of m#0 with the error; otherwise it leaves
				// m#0 untouched.
				switch vm.Mem[args[1]].T {
				case values.BOOL:
					if !(vm.Mem[args[1]].V.(bool)) {
						tokNumber := uint32(vm.Mem[args[2]].V.(int))
						errorInfo := vm.ValidationErrors[args[3]]
						vm.Mem[args[0]] = vm.makeError("vm/validation/fail", tokNumber,
							errorInfo.Condition, errorInfo.Type, errorInfo.Tok, errorInfo.Value)
						if len(vm.callstack) == stackHeight {
							return
						}
						loc = vm.callstack[len(vm.callstack)-1]
						vm.callstack = vm.callstack[0 : len(vm.callstack)-1]
					}
				case values.ERROR:
					vm.Mem[args[0]] = vm.Mem[args[1]]
				default:
					tokNumber := uint32(vm.Mem[args[2]].V.(int))
					errorInfo := vm.ValidationErrors[args[3]]
					vm.Mem[args[0]] = vm.makeError("vm/validation/bool", tokNumber,
						errorInfo.Condition, errorInfo.Type, errorInfo.Tok,
						vm.DescribeType(vm.Mem[args[1]].T, LITERAL, 0), vm.Mem[args[1]], errorInfo.Tok)
					if len(vm.callstack) == stackHeight {
						return
					}
					loc = vm.callstack[len(vm.callstack)-1]
					vm.callstack = vm.callstack[0 : len(vm.callstack)-1]
				}
			case Chrf: // Check reference variable (dst mem)
				// At the end of executing a command, if it has reference variables, if we have inserted an
				// error into any of the reference variables, we must return the first of these errors instead
				// of `OK`. m#0 is the return location of the command; m#1 contains the reference variable.
				if vm.Mem[vm.Mem[args[1]].V.(uint32)].T == values.ERROR {
					vm.Mem[args[0]] = vm.Mem[vm.Mem[args[1]].V.(uint32)]
				}
			case Clon: // Clones of type  (dst mem)
				// Implements `clones{T}`.
				if vm.Mem[args[1]].T != values.TYPE {
					vm.Mem[args[0]] = vm.makeError("vm/clones/type", args[2], args[1])
					break Switch
				}
				abType := values.AbstractType{}
				for _, v := range vm.Mem[args[1]].V.(values.AbstractType).Types {
					clones := vm.ConcreteTypeInfo[v].IsClonedBy()
					abType = abType.Union(clones)
				}
				vm.Mem[args[0]] = values.Value{values.TYPE, abType}
			case CoSn: // Construct snippet from arguments (dst mem)
				vm.Mem[args[0]] = values.Value{values.SNIPPET, values.Snippet{vm.Mem[args[1]].V.([]values.Value), nil}}
			case ConL: // Append element to list  (dst mem mem)
				// Appends an element to a list, i.e. implements `L & x` where `L` is a list.
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(vector.Vector).Conj(vm.Mem[args[2]])}
			case ConS: // Add element to set (dst mem mem)
				// Adds an element to a list, i.e. implements `S & x` where `L` is a set.
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(values.Set).Add(vm.Mem[args[2]])}
			case Cpnt: // Codepoint of rune (dst mem)
				// Converts a rune into its Unicode code point, represented as an integer.
				vm.Mem[args[0]] = values.Value{values.INT, int(vm.Mem[args[1]].V.(rune))}
			case Cv1T: // Convert element to tuple (dst mem)
				// Converts v#1 to the tuple containing v#1.
				vm.Mem[args[0]] = values.Value{values.TUPLE, []values.Value{vm.Mem[args[1]]}}
			case CvTT: // Create tuple (dst tup)
				// Takes v#2 ... v#n and returns a tuple consisting of those elements.
				slice := []values.Value{}
				for i := 1; i < len(args); i++ {
					if vm.Mem[args[i]].T == values.TUPLE {
						slice = append(slice, vm.Mem[args[i]].V.([]values.Value)...)
					} else {
						slice = append(slice, vm.Mem[args[i]])
					}
				}
				vm.Mem[args[0]] = values.Value{values.TUPLE, slice}
			case Diif: // Divide ints as float  (dst mem mem tok)
				// Divides two integers as a float, i.e. it implements `m / n` where `m` and `n` are 
				// integers. It returns an error constructed from token number n#3 if v#2 is 0. 
				divisor := vm.Mem[args[2]].V.(int)
				if divisor == 0 {
					vm.Mem[args[0]] = vm.makeError("vm/div/zero/a", args[3])
				} else {
					vm.Mem[args[0]] = values.Value{values.FLOAT, float64(vm.Mem[args[1]].V.(int)) / float64(divisor)}
				}
			case Divf: // Divide floats  (dst mem mem tok)
				// Divides two floats, returning a float.
				// It returns an error constructed from token number n#3 if v#2 is 0.0. 
				divisor := vm.Mem[args[2]].V.(float64)
				if divisor == 0 {
					vm.Mem[args[0]] = vm.makeError("vm/div/zero/b", args[3])
				} else {
					vm.Mem[args[0]] = values.Value{values.FLOAT, vm.Mem[args[1]].V.(float64) / divisor}
				}
			case Divi: // Divide ints  (dst mem mem tok)
				// Divides two integers and returns an integer, i.e. it implements `m div n`.
				// It returns an error constructed from token number n#3 if v#2 is 0. 
				divisor := vm.Mem[args[2]].V.(int)
				if divisor == 0 {
					vm.Mem[args[0]] = vm.makeError("vm/div/zero/c", args[3])
				} else {
					vm.Mem[args[0]] = values.Value{values.INT, vm.Mem[args[1]].V.(int) / vm.Mem[args[2]].V.(int)}
				}
			case Dvfi: // Divide float by int (dst mem mem tok)
				// Divides a float by an int and returns a float.
				// It returns an error constructed from token number n#3 if v#2 is 0. 
				divisor := vm.Mem[args[2]].V.(int)
				if divisor == 0 {
					vm.Mem[args[0]] = vm.makeError("vm/div/zero/d", args[3])
				} else {
					vm.Mem[args[0]] = values.Value{values.FLOAT, vm.Mem[args[1]].V.(float64) / float64(divisor)}
				}
			case Dvif: // Divide int by float (dst mem mem tok)
				// Divides an int by a float and returns a float.
				// It returns an error constructed from token number n#3 if v#2 is 0.0.
				divisor := vm.Mem[args[2]].V.(float64)
				if divisor == 0 {
					vm.Mem[args[0]] = vm.makeError("vm/div/zero/e", args[3])
				} else {
					vm.Mem[args[0]] = values.Value{values.FLOAT, float64(vm.Mem[args[1]].V.(int)) / divisor}
				}
			case Dofn: // Apply lambda function (dst mem tup)
				// Applies the function v#1 to the values in the tuple.
				lambda := vm.Mem[args[1]].V.(Lambda)
				// The case where the lambda is from a Go function.
				if lambda.Gocode != nil {
					goArgs := []reflect.Value{}
					for _, pfMemLoc := range args[2:] {
						pfArg := vm.Mem[pfMemLoc]
						goArg, ok := vm.pipefishToGo(pfArg)
						if !ok {
							vm.Mem[args[0]] = values.Value{values.ERROR, err.CreateErr("vm/func/go", lambda.Tok, goArg)} // If the conversion failed, the goArg will be the value it couldn't convert.
							break Switch
						}
						goArgs = append(goArgs, reflect.ValueOf(goArg))
					}
					goResultValues := lambda.Gocode.Call(goArgs)
					var doctoredValues any
					if len(goResultValues) == 1 {
						doctoredValues = goResultValues[0].Interface()
					} else {
						elements := make([]any, 0, len(goResultValues))
						for _, v := range goResultValues {
							elements = append(elements, v.Interface())
						}
						doctoredValues = goTuple(elements)
					}
					val := vm.goToPipefish(reflect.ValueOf(doctoredValues))
					if val.T == 0 {
						payload := val.V.([]any)
						newError := err.CreateErr(payload[0].(string), vm.Mem[args[1]].V.(*err.Error).Token, payload[1:]...)
						vm.Mem[args[0]] = values.Value{values.ERROR, newError}
						break
					}
					if val.T == values.ERROR {
						val.V.(*err.Error).Token = vm.Mem[args[1]].V.(*err.Error).Token
					}
					vm.Mem[args[0]] = val
					break Switch
				}
				// The normal case.
				// The code here is repeated with a few twists in a very non-DRY way in the go handler and any changes necessary here will probably need to be copied there.
				if len(args)-2 != len(lambda.Sig) { // TODO: variadics.
					vm.Mem[args[0]] = values.Value{values.ERROR, err.CreateErr("vm/func/args", lambda.Tok)}
					break Switch
				}
				for i := 0; i < int(lambda.CapturesEnd-lambda.CapturesStart); i++ {
					vm.Mem[int(lambda.CapturesStart)+i] = lambda.Captures[i]
				}
				for i := 0; i < int(lambda.ParametersEnd-lambda.CapturesEnd); i++ {
					vm.Mem[int(lambda.CapturesEnd)+i] = vm.Mem[args[2+i]]
				}
				success := true
				if lambda.Sig != nil {
					for i, abType := range lambda.Sig { // TODO --- as with other such cases there will be a threshold at which linear search becomes inferior to binary search and we should find out what it is.
						success = false
						if abType.Types == nil { // Used for `any?`.
							success = true
							continue
						} else {
							for _, ty := range abType.Types {
								if ty == vm.Mem[int(lambda.CapturesEnd)+i].T {
									success = true
									if vm.Mem[int(lambda.CapturesEnd)+i].T == values.STRING && len(vm.Mem[int(lambda.CapturesEnd)+i].V.(string)) > abType.Len() {
										success = false
									}
								}
							}
						}
						if !success {
							vm.Mem[args[0]] = values.Value{values.ERROR, err.CreateErr("vm/func/types", lambda.Tok)}
							break Switch
						}
					}
				}
				vm.run(lambda.AddressToCall, ctx, cancel)
				vm.Mem[args[0]] = vm.Mem[lambda.ResultLocation]
			case Dref: // Dereference ref variable (dst mem)
				// Puts the contents of the reference variable in m#1 into m#0.
				vm.Mem[args[0]] = vm.Mem[vm.Mem[args[1]].V.(uint32)]
			case Equb: // Boolean comparison with == (dst mem mem)
				// Tests if two booleans are equal.
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(bool) == vm.Mem[args[2]].V.(bool)}
			case Equf: // Float comparison with == (dst mem mem)
				// Tests if two floats are equal.
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(float64) == vm.Mem[args[2]].V.(float64)}
			case Equi: // Integer comparison with == (dst mem mem)
				// Tests if two ints are equal.
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(int) == vm.Mem[args[2]].V.(int)}
			case Equs: // String comparison with == (dst mem mem)
				// Tests if two strings are equal
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(string) == vm.Mem[args[2]].V.(string)}
			case Equt: // Type comparison with == (dst mem mem)
				// Tests if two types are equal
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(values.AbstractType).Equals(vm.Mem[args[2]].V.(values.AbstractType))}
			case Eqxx: // Comparison with == (dst mem mem tok)
				// Tests if two values are equal. If they are not comparable, we return an error
				// based on token number n#3.
				if vm.Mem[args[1]].T == values.ERROR {
					vm.Mem[args[0]] = vm.Mem[args[1]]
					break
				}
				if vm.Mem[args[2]].T == values.ERROR {
					vm.Mem[args[0]] = vm.Mem[args[2]]
					break
				}
				if vm.Mem[args[1]].T != vm.Mem[args[2]].T {
					vm.Mem[args[0]] = vm.makeError("vm/equals/type", args[3], args[1], args[2], vm.DescribeType(vm.Mem[args[1]].T, LITERAL, 0), vm.DescribeType(vm.Mem[args[2]].T, LITERAL, 0))
				} else {
					vm.Mem[args[0]] = values.Value{values.BOOL, vm.equals(vm.Mem[args[1]], vm.Mem[args[2]])}
				}
			case Eval: // Eval (dst mem num)
				// This evaluates the string v#1 using evaluator number n#2.
				vm.Mem[args[0]] = vm.Evaluators[args[2]](vm.Mem[args[1]].V.(string))
			case Extn: // External service call (dst num num mem mem tup)
				// Operands are: 
				//     n#1 : the number of the external service to call
				//     n#2 : whether the function being called is a prefix, infix, postfix or unfix.
				//     v#2 : the remainder of the namespace of the function as a string
				//     v#3 : the name of the function as a string
				//     #4 : a tuple of the locations of the arguments we wish to pass.
				externalOrdinal := args[1]
				operatorType := args[2]
				remainingNamespace := vm.Mem[args[3]].V.(string)
				name := vm.Mem[args[4]].V.(string)
				argLocs := args[5:]
				lastWasBling := false
				var buf strings.Builder
				if operatorType == PREFIX || operatorType == UNFIX {
					buf.WriteString(remainingNamespace)
					buf.WriteString(name)
					lastWasBling = true
				}
				if operatorType == PREFIX {
					if len(argLocs) == 0 {
						buf.WriteString("(")
					}
					lastWasBling = len(argLocs) > 0
				}
				if operatorType == INFIX || operatorType == SUFFIX {
					buf.WriteString("(")
				}
				for i, loc := range argLocs {
					serializedValue := vm.Literal(vm.Mem[loc], 0)
					if operatorType == INFIX && vm.Mem[loc].T == values.BLING && serializedValue == name { // Then we need to attach the namespace to the operator.
						buf.WriteString(remainingNamespace)
					}
					if vm.Mem[loc].T == values.BLING {
						if !lastWasBling {
							buf.WriteString(")")
						}
						buf.WriteString(" ")
						buf.WriteString(serializedValue)
						lastWasBling = true
						continue
					}
					// So it's non-bling
					if lastWasBling {
						buf.WriteString(" (")
					} else {
						if i > 0 {
							buf.WriteString(", ")
						}
					}
					lastWasBling = false
					buf.WriteString(serializedValue)
				}
				if !lastWasBling {
					buf.WriteString(")")
				}
				if operatorType == SUFFIX {
					buf.WriteString(remainingNamespace)
					buf.WriteString(name)
				}
				vm.Mem[args[0]] = vm.ExternalCallHandlers[externalOrdinal].Evaluate(buf.String())
			case Flpp: // Pop peek flags ()
				vm.PopPeeks()
			case Flps: // Push peek flags (mem)
				// v#0 will be of internal type PEEK_FLAGS.
				vm.PeekStack = append(vm.PeekStack, vm.Mem[args[0]].V.(map[string]bool))
			case Flti: // Float from int (dst mem)
				vm.Mem[args[0]] = values.Value{values.FLOAT, float64(vm.Mem[args[1]].V.(int))}
			case Flts: // Float from string (dst mem tok)
				// Token number n#2 is used to make an error if the conversion fails.
				i, err := strconv.ParseFloat(vm.Mem[args[1]].V.(string), 64)
				if err != nil {
					vm.Mem[args[0]] = vm.makeError("vm/string/float", args[2], args[1])
				} else {
					vm.Mem[args[0]] = values.Value{values.FLOAT, i}
				}
			case Gofn: // Call Go function (dst mem gfn tup)
				// Operands are :
				//     m#1 : contains an error which we will doctor before (if necessary) returning it.
				//     n#2 : the number of the Go function we want to call.
				//     #3 : a tuple of the locations of the arguments we want to pass to the function.
				F := vm.GoFns[args[2]]
				goTpl := make([]reflect.Value, 0, len(args))
				for _, v := range args[3:] { // TODO --- how can this be right? Surely they should be stored in a TUPLE.
					el := vm.Mem[v]
					goVal, ok := vm.pipefishToGo(el)
					if !ok {
						newError := err.CreateErr("vm/pipefish/type", vm.Mem[args[1]].V.(*err.Error).Token, vm.DescribeType(el.T, LITERAL, 0))
						newError.Values = []values.Value{el}
						vm.Mem[args[0]] = values.Value{values.ERROR, newError}
						break Switch
					}
					goTpl = append(goTpl, reflect.ValueOf(goVal))
				}
				var goResultValues []reflect.Value
				goResultValues = F.Code.Call(goTpl)
				var doctoredValues any
				if len(goResultValues) == 1 {
					doctoredValues = goResultValues[0].Interface()
				} else {
					elements := make([]any, 0, len(goResultValues))
					for _, v := range goResultValues {
						elements = append(elements, v.Interface())
					}
					doctoredValues = goTuple(elements)
				}
				val := vm.goToPipefish(reflect.ValueOf(doctoredValues))
				if val.T == 0 {
					payload := val.V.([]any)
					newError := err.CreateErr(payload[0].(string), vm.Mem[args[1]].V.(*err.Error).Token, payload[1:]...)
					vm.Mem[args[0]] = values.Value{values.ERROR, newError}
					break Switch
				}
				vm.Mem[args[0]] = val
			case Gsql: // Get from SQL (dst mem mem mem mem num tok)
				// This returns an error or `OK` in m#0, the SQL data being put in the reference variable v#1.
				// Operands are :
				//     v#1 : the address of the reference variable: where we put what we get from SQL.
				//     v#2 : the desired type of the result
				// 	v#3 : the database connection
				// 	v#4 : the snippet of SQL
				// 	n#5 : 0 for `get as`, 1 for `get like`.
				// 	n#6 : the number of a token for emitting an error if required.
				rType := vm.Mem[args[2]].V.(values.AbstractType)
				if rType.Len() != 1 {
					vm.Mem[args[0]] = vm.makeError("vm/sql/abstract/c", args[5], vm.DescribeAbstractType(vm.Mem[args[2]].V.(values.AbstractType), LITERAL, 0))
					break Switch
				}
				cType := rType.Types[0]
				sqlObj := vm.Mem[args[3]].V.(*sql.DB)

				snippet := vm.Mem[args[4]].V.(values.Snippet).Data
				buf := strings.Builder{}
				vals := make([]values.Value, 0, len(snippet)/2)
				for i, v := range snippet {
					if i%2 == 0 {
						buf.WriteString(v.V.(string))
					} else {
						vals = append(vals, v)
						buf.WriteString("$")
						buf.WriteString(strconv.Itoa(1 + i/2))
					}
				}
				result := vm.evalGetSQL(sqlObj, cType, buf.String(), vals, args[5], args[6], ctx)
				vm.Mem[args[1]] = result
				if result.T == values.ERROR {
					vm.Mem[args[0]] = result
				} else {
					vm.Mem[args[0]] = values.OK
				}
			case Gtef: // Float comparison with >= (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(float64) >= vm.Mem[args[2]].V.(float64)}
			case Gtei: // Int comparison with >= (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(int) >= vm.Mem[args[2]].V.(int)}
			case Gthf: // Float comparison with > (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(float64) > vm.Mem[args[2]].V.(float64)}
			case Gthi: // Int comparison with > (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].V.(int) > vm.Mem[args[2]].V.(int)}
			case IctS: // Intersection of sets (dst mem mem)
				leftSet := vm.Mem[args[1]].V.(values.Set)
				result := leftSet.Intersect(vm.Mem[args[2]].V.(values.Set))
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, result}
			case IdxL: // Index list  (dst mem mem tok)
				// v#1 is the list, v#2 is an integer, and n#3 is the number of a token to make an error
				// in the case that v#2 is out of bounds.
				vec := vm.Mem[args[1]].V.(vector.Vector)
				ix := vm.Mem[args[2]].V.(int)
				val, ok := vec.Index(ix)
				if !ok {
					vm.Mem[args[0]] = vm.makeError("vm/index/list", args[3], ix, vec.Len(), args[1], args[2])
				} else {
					vm.Mem[args[0]] = val.(values.Value)
				}
			case Idxp: // Index pair  (dst mem mem tok)
				// v#1 is the pair, v#2 is an integer, and n#3 is the number of a token to make an error
				// in the case that v#2 is out of bounds.
				pair := vm.Mem[args[1]].V.([]values.Value)
				ix := vm.Mem[args[2]].V.(int)
				ok := ix == 0 || ix == 1
				if ok {
					vm.Mem[args[0]] = pair[ix]
				} else {
					vm.Mem[args[0]] = vm.makeError("vm/index/pair", args[3], ix)
				}
			case Idxs: // Index string (dst mem mem tok)
				// v#1 is the string, v#2 is an integer, and n#3 is the number of a token to make an error
				// in the case that v#2 is out of bounds.
				str := vm.Mem[args[1]].V.(string)
				ix := vm.Mem[args[2]].V.(int)
				ok := 0 <= ix && ix < len(str)
				if ok {
					val := values.Value{values.RUNE, rune(str[ix])}
					vm.Mem[args[0]] = val
				} else {
					vm.Mem[args[0]] = vm.makeError("vm/index/string", args[3], ix, len(str), args[1], args[2])
				}
			case IdxT: // Index tuple (dst mem mem tok)
				// v#1 is the tuple, v#2 is an integer, and n#3 is the number of a token to make an error
				// in the case that v#2 is out of bounds.
				tuple := vm.Mem[args[1]].V.([]values.Value)
				ix := vm.Mem[args[2]].V.(int)
				ok := 0 <= ix && ix < len(tuple)
				if ok {
					vm.Mem[args[0]] = tuple[ix]
				} else {
					vm.Mem[args[0]] = vm.makeError("vm/index/tuple", args[3], ix, len(tuple), args[1], args[2])
				}
			case Inpt: // Input from keyboard (dst mem mem)
				// v#1 is of type `terminal.Keyboard` with one field consisting of the prompt. #v2 is a 
				// boolean saying whether the input should be masked for privacy.
				temp := vm.InHandle
				if vm.Mem[args[2]].V.(bool) {
					vm.InHandle = &MaskedInHandler{vm.Mem[args[1]].V.([]values.Value)[0].V.(string), cancel}
				} else {
					vm.InHandle = &StandardInHandler{vm.Mem[args[1]].V.([]values.Value)[0].V.(string), cancel}
				}
				response := vm.InHandle.Get()
				vm.InHandle = temp
				vm.Mem[vm.Mem[args[0]].V.(uint32)] = values.Value{values.STRING, response}
			case Inte: // Integer from enum (dst mem)
				vm.Mem[args[0]] = values.Value{values.INT, vm.Mem[args[1]].V.(int)}
			case Intf: // Integer from float (dst mem)
				vm.Mem[args[0]] = values.Value{values.INT, int(vm.Mem[args[1]].V.(float64))}
			case Ints: // Integer from string  (dst mem tok)
				// n#2 is the number of a token to make an error if conversion fails.
				i, err := strconv.Atoi(vm.Mem[args[1]].V.(string))
				if err != nil {
					vm.Mem[args[0]] = vm.makeError("vm/string/int", args[2], args[1])
				} else {
					vm.Mem[args[0]] = values.Value{values.INT, i}
				}
			case InxL: // Is element in list (dst mem mem)
				x := vm.Mem[args[1]]
				L := vm.Mem[args[2]].V.(vector.Vector)
				i := 0
				vm.Mem[args[0]] = values.Value{values.BOOL, false}
				for el, ok := L.Index(i); ok; {
					if x.T == el.(values.Value).T {
						if vm.equals(x, el.(values.Value)) {
							vm.Mem[args[0]] = values.Value{values.BOOL, true}
							break
						}
					}
					i++
					el, ok = L.Index(i)
				}
			case InxS: // Hard-index snippet (dst mem num)
				// Returns element number n#2 of snippet v#1.
				x := vm.Mem[args[1]]
				S := vm.Mem[args[2]].V.(values.Set)
				if S.Contains(x) {
					vm.Mem[args[0]] = values.Value{values.BOOL, true}
				} else {
					vm.Mem[args[0]] = values.Value{values.BOOL, false}
				}
			case InxT: // Is element in tuple (dst mem mem)
				x := vm.Mem[args[1]]
				T := vm.Mem[args[2]].V.([]values.Value)
				vm.Mem[args[0]] = values.Value{values.BOOL, false}
				for _, el := range T {
					if x.T == el.T {
						if vm.equals(x, el) {
							vm.Mem[args[0]] = values.Value{values.BOOL, true}
							break
						}
					}
				}
			case Inxt: // Is element in type (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.BOOL, false}
				for _, t := range vm.Mem[args[2]].V.(values.AbstractType).Types {
					if vm.Mem[args[1]].T == t {
						vm.Mem[args[0]] = values.Value{values.BOOL, true}
					}
				}
			case Itgk: // Get key from iterator (dst mem)
				vm.Mem[args[0]] = vm.Mem[args[1]].V.(Iterator).GetKey()
			case Itkv: // Get key and value from iterator (dst dst mem)
				vm.Mem[args[0]], vm.Mem[args[1]] = vm.Mem[args[2]].V.(Iterator).GetKeyValuePair()
			case Itgv: // Get value from iterator (dst mem)
				vm.Mem[args[0]] = vm.Mem[args[1]].V.(Iterator).GetValue()
			case Itor: // Integer to rune (dst mem)
				vm.Mem[args[0]] = values.Value{values.RUNE, rune(vm.Mem[args[1]].V.(int))}
			case IxSn: // Index snippet (dst mem mem tok)
				// v#1 is the tuple, v#2 is an integer, and n#3 is the number of a token to make an error
				// in the case that v#2 is out of bounds.
				ix := vm.Mem[args[2]].V.(int)
				if ix < 0 || ix >= len(vm.Mem[args[1]].V.(values.Snippet).Data) {
					vm.Mem[args[0]] = vm.makeError("vm/index/s", args[3], ix)
				} else {
					vm.Mem[args[0]] = vm.Mem[args[1]].V.(values.Snippet).Data[ix]
				}
			// This is emitted by `function_call` and the typechecking logic and so the index should
			// be correct and no bounds-checking is required.
			case IxTn: // Hard-index tuple (dst mem num)
				// v#1 is the tuple, and we index it by n#2.
				vm.Mem[args[0]] = (vm.Mem[args[1]].V.([]values.Value))[args[2]]
			case IxXx: // Index value by value (dst mem mem tok)
				// In the case where at compile time we can't determine the types of one or other or both
				// of v#1 and v#2. The token number n#3 can be used to create an error if the index is the 
				// wrong type or out of bounds.
				container := vm.Mem[args[1]]
				if container.T == values.ERROR {
					vm.Mem[args[0]] = container
					break Switch
				}
				index := vm.Mem[args[2]]
				if index.T == values.ERROR {
					vm.Mem[args[0]] = index
					break Switch
				}
				indexType := index.T
				if cloneInfo, ok := vm.ConcreteTypeInfo[indexType].(CloneType); ok {
					indexType = cloneInfo.Parent
				}
				containerType := container.T
				if cloneInfo, ok := vm.ConcreteTypeInfo[containerType].(CloneType); ok {
					containerType = cloneInfo.Parent
				}
				if indexType == values.PAIR { // Then we're slicing.
					ix := vm.Mem[args[2]].V.([]values.Value)
					if ix[0].T != values.INT {
						vm.Mem[args[0]] = vm.makeError("vm/index/a", args[3], vm.DescribeType(ix[0].T, LITERAL, 0))
						break Switch
					}
					if ix[1].T != values.INT {
						vm.Mem[args[0]] = vm.makeError("vm/index/b", args[3], vm.DescribeType(ix[1].T, LITERAL, 0))
						break Switch
					}
					if ix[0].V.(int) < 0 {
						vm.Mem[args[0]] = vm.makeError("vm/index/c", args[3], ix[0].V.(int))
						break Switch
					}
					if ix[1].V.(int) < ix[0].V.(int) {
						vm.Mem[args[0]] = vm.makeError("vm/index/d", args[3], ix[0].V.(int), ix[1].V.(int))
						break Switch
					}
					// We switch on the type of the lhs.
					switch containerType {
					case values.LIST:
						vec := vm.Mem[args[1]].V.(vector.Vector)
						if ix[1].V.(int) > vec.Len() {
							vm.Mem[args[0]] = vm.makeError("vm/index/e", args[3], ix[1].V.(int), vec.Len(), args[1], args[2])
							break Switch
						}
						vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vec.SubVector(ix[0].V.(int), ix[1].V.(int))}
					case values.STRING:
						str := container.V.(string)
						ix := index.V.([]values.Value)
						if ix[1].V.(int) > len(str) {
							vm.Mem[args[0]] = vm.makeError("vm/index/f", args[3], ix[1].V.(int), len(str), args[1], args[2])
							break Switch
						}
						vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, str[ix[0].V.(int):ix[1].V.(int)]}
					case values.TUPLE:
						tup := container.V.([]values.Value)
						if ix[1].V.(int) > len(tup) {
							vm.Mem[args[0]] = vm.makeError("vm/index/r", args[3], ix[1].V.(int), len(tup))
							break Switch
						}
						vm.Mem[args[0]] = values.Value{values.TUPLE, tup[ix[0].V.(int):ix[1].V.(int)]}
					default:
						vm.Mem[args[0]] = vm.makeError("vm/index/g", args[3], vm.DescribeType(container.T, LITERAL, 0))
						break Switch
					}
				} else {
					// Otherwise it's not a slice. We switch on the type of the lhs.
					typeInfo := vm.ConcreteTypeInfo[containerType]
					if typeInfo.IsStruct() {
						if vm.Mem[args[2]].T != values.LABEL {
							vm.Mem[args[0]] = vm.makeError("vm/index/label", args[3], args[2], vm.DescribeType(vm.Mem[args[2]].T, LITERAL, 0))
							break Switch
						}
						ix := typeInfo.(StructType).Resolve(vm.Mem[args[2]].V.(int))
						if ix == -1 {
							vm.Mem[args[0]] = vm.makeError("vm/index/t", args[3], typeInfo.(StructType).Name, vm.Labels[vm.Mem[args[2]].V.(int)])
						} else {
							vm.Mem[args[0]] = vm.Mem[args[1]].V.([]values.Value)[ix]
						}
						break Switch
					}
					if containerType == values.MAP {
						mp := container.V.(values.Map)
						ix := vm.Mem[args[2]]
						result, ok := mp.Get(ix)
						if !ok {
							vm.Mem[args[0]] = vm.makeError("vm/index/h", args[3], vm.DefaultDescription(vm.Mem[args[2]]), args[1], args[2])
						} else {
							vm.Mem[args[0]] = result
						}
						break
					}
					if indexType != values.INT {
						vm.Mem[args[0]] = vm.makeError("vm/index/i", args[3], vm.DescribeType(vm.Mem[args[1]].T, LITERAL, 0), vm.DescribeType(vm.Mem[args[2]].T, LITERAL, 0), args[1], args[2])
						break
					}
					ty := container.T
					if cloneInfo, ok := vm.ConcreteTypeInfo[container.T].(CloneType); ok {
						ty = cloneInfo.Parent
					}
					switch ty {
					case values.LIST:
						vec := container.V.(vector.Vector)
						ix := index.V.(int)
						val, ok := vec.Index(ix)
						if !ok {
							vm.Mem[args[0]] = vm.makeError("vm/index/j", args[3], ix, vec.Len(), args[1], args[2])
						} else {
							vm.Mem[args[0]] = val.(values.Value)
						}
						break Switch
					case values.PAIR:
						pair := container.V.([]values.Value)
						ix := index.V.(int)
						ok := ix == 0 || ix == 1
						if ok {
							vm.Mem[args[0]] = pair[ix]
						} else {
							vm.Mem[args[0]] = vm.makeError("vm/index/k", args[3], ix)
						}
						break Switch
					case values.SNIPPET:
						snippetData := container.V.(values.Snippet).Data
						ix := index.V.(int)
						ok := 0 <= ix && ix < len(snippetData)
						if ok {
							vm.Mem[args[0]] = snippetData[ix]
						} else {
							vm.Mem[args[0]] = vm.makeError("vm/index/s", args[3], ix, len(snippetData), args[1], args[2])
						}
						break Switch
					case values.STRING:
						str := container.V.(string)
						ix := index.V.(int)
						ok := 0 <= ix && ix < len(str)
						if ok {
							val := values.Value{values.RUNE, rune(str[ix])}
							vm.Mem[args[0]] = val
						} else {
							vm.Mem[args[0]] = vm.makeError("vm/index/l", args[3], ix, len(str), args[1], args[2])
						}
						break Switch
					case values.TUPLE:
						tuple := container.V.([]values.Value)
						ix := index.V.(int)
						ok := 0 <= ix && ix < len(tuple)
						if ok {
							vm.Mem[args[0]] = tuple[ix]
						} else {
							vm.Mem[args[0]] = vm.makeError("vm/index/m", args[3], ix, len(tuple), args[1], args[2])
						}
						break Switch
					default:
						vm.Mem[args[0]] = vm.makeError("vm/index/q", args[3], vm.DescribeType(vm.Mem[args[1]].T, LITERAL, 0), vm.DescribeType(vm.Mem[args[2]].T, LITERAL, 0))
						break Switch
					}
				}
			case IxZl: // Index struct by label (dst mem mem tok)
				// v#1 is the struct, v#2 is a label, and n#3 is the number of a token to make an error
				// in the case that v#1 has no field labeled by v#2.
				typeInfo := vm.ConcreteTypeInfo[vm.Mem[args[1]].T].(StructType)
				ix := typeInfo.Resolve(vm.Mem[args[2]].V.(int))
				if ix == -1 {
					vm.Mem[args[0]] = vm.makeError("vm/index/u", args[3], vm.DescribeType(vm.Mem[args[1]].T, LITERAL, 0), vm.DefaultDescription(vm.Mem[args[2]]))
					break Switch
				}
				vm.Mem[args[0]] = vm.Mem[args[1]].V.([]values.Value)[ix]
			case IxZn: // Hard-index struct (dst mem num)
				// v#1 is the struct and n#2 is the number of the field we want to index, determined at
				// compile-time.
				vm.Mem[args[0]] = vm.Mem[args[1]].V.([]values.Value)[args[2]]
			case Jmp: // Jump (loc)
				loc = args[0]
				continue
			case Json: // Json to Pipefish (dst mem mem num tok)
				// Operands are :
				//     v#1 : a string containing the JSON.
				//     v#2 : the type to convert to.
				//     n#3 : 0 or 1 to indicate whether we are converting "like" or "as".
				//     n#4 : the number of a token for constructing an error in the case the conversion fails.
				vm.Mem[args[0]] = vm.jsonToPf(vm.Mem[args[1]].V.(string), vm.Mem[args[2]].V.(values.AbstractType), args[3] == 0, args[4], ctx, cancel)
			case Jsr: // Jump to subroutine (loc)
				// Pushes the location we're jumping from onto the stack, so that `rtn` will return to just after
				// the jump.
				vm.callstack = append(vm.callstack, loc)
				loc = args[0]
				continue
			case KeyM: // Keys of map (dst mem)
				// Returned as a list.
				vm.Mem[args[0]] = values.Value{values.LIST, vm.Mem[args[1]].V.(values.Map).KeysAsVector()}
			case KeyZ: // Keys of struct (dst mem)
				// Returned as a list containing the labels.
				result := vector.Empty
				for _, labelNumber := range vm.ConcreteTypeInfo[vm.Mem[args[1]].T].(StructType).LabelNumbers {
					result = result.Conj(values.Value{values.LABEL, labelNumber})
				}
				vm.Mem[args[0]] = values.Value{values.LIST, result}
			case Lbls: // Label from string (dst mem tok)
				// Returns token number n#2 if the conversion fails.
				stringToConvert := vm.Mem[args[1]].V.(string)
				labelNo, ok := vm.FieldLabelsInMem[stringToConvert]
				if ok {
					vm.Mem[args[0]] = vm.Mem[labelNo]
				} else {
					vm.Mem[args[0]] = vm.makeError("vm/label/exists", args[2], stringToConvert)
				}
			case LenL: // Length of list (dst mem)
				vm.Mem[args[0]] = values.Value{values.INT, vm.Mem[args[1]].V.(vector.Vector).Len()}
			case LenM: // Length of map (dst mem)
				vm.Mem[args[0]] = values.Value{values.INT, vm.Mem[args[1]].V.(values.Map).Len()}
			case Lens: // Length of string (dst mem)
				vm.Mem[args[0]] = values.Value{values.INT, len(vm.Mem[args[1]].V.(string))}
			case LenS: // Length of set (dst mem)
				vm.Mem[args[0]] = values.Value{values.INT, vm.Mem[args[1]].V.(values.Set).Len()}
			case LenT: // Length of tuple (dst mem)
				vm.Mem[args[0]] = values.Value{values.INT, len(vm.Mem[args[1]].V.([]values.Value))}
			case List: // List from tuple (dst mem)
				list := vector.Empty
				if vm.Mem[args[1]].T == values.TUPLE {
					for _, v := range vm.Mem[args[1]].V.([]values.Value) {
						list = list.Conj(v)
					}
				} else {
					list = list.Conj(vm.Mem[args[1]])
				}
				vm.Mem[args[0]] = values.Value{values.LIST, list}
			case Litx: // Literal of value (dst mem num tok)
				// Operands :
				//     v#1 is the value.
				//     n#2 is the number of the compiler to generate the literal.
				//     n#3 is the number of a token for error-generation if the value has no literal representation.
				vm.Mem[args[0]] = values.Value{values.STRING, vm.Literal(vm.Mem[args[1]], args[2])}
			case LnSn: // Length of snippet (dst mem)
				vm.Mem[args[0]] = values.Value{values.INT, len(vm.Mem[args[1]].V.(values.Snippet).Data)}
			case Logn: // Turn logging off ()
				vm.logging = false
			case Logy: // Turn logging on ()
				vm.logging = true
			case MkEn: // Enum element from int (dst typ mem tok)
				// Makes an enum of type number n#1 from an integer v#2, using token n#3 to return an error if
				// v#2 is out of bounds.
				info := vm.ConcreteTypeInfo[args[1]].(EnumType)
				ix := vm.Mem[args[2]].V.(int)
				ok := 0 <= ix && ix < len(info.ElementNames)
				if ok {
					vm.Mem[args[0]] = values.Value{values.ValueType(args[1]), ix}
				} else {
					vm.Mem[args[0]] = vm.makeError("vm/enum", args[3], info.GetName(LITERAL), ix)
				}
			case Mker: // Error from string (dst mem tok)
				vm.Mem[args[0]] = values.Value{values.ERROR, &err.Error{ErrorId: "vm/user", Message: vm.Mem[args[1]].V.(string), Token: vm.Tokens[args[2]]}}
			case Mkfn: // Make lambda (dst lfc)
				// Here n#1 is the number of a lambda factory which knows how to make the lambda.
				lf := vm.LambdaFactories[args[1]]
				newLambda := *lf.Model
				newLambda.Captures = make([]values.Value, len(lf.CaptureLocations))
				for i, v := range lf.CaptureLocations {
					val := vm.Mem[v]
					if val.T == values.THUNK {
						vm.run(val.V.(uint32), ctx, cancel)
						val = vm.Mem[v]
					}
					newLambda.Captures[i] = val
				}
				vm.Mem[args[0]] = values.Value{values.FUNC, newLambda}
			case Mkit: // Make iterator (dst mem num tok)
				// v#1 is the range of the iterator, n#2 is 0 or 1 according to whether the iterator  doesn't or
				// does only return keys, and token number n#3 is used to create a runtime error if the range
				// is invalid.
				vm.Mem[args[0]] = vm.NewIterator(vm.Mem[args[1]], args[2] == 1, args[3])
			case Mkmp: // Make map (dst mem tok)
				// Constructs a map from a tuple value v#1, using token n#2 to create an error if the
				// elements of the tuple have the wrong type, e.g. there's an unhashable key.
				result := values.Map{}
				for _, p := range vm.Mem[args[1]].V.([]values.Value) {
					k := p.V.([]values.Value)[0]
					v := p.V.([]values.Value)[1]
					if !((values.NULL <= k.T && k.T < values.PAIR) || vm.ConcreteTypeInfo[k.T].IsEnum() || // TODO, we can just have a simple filter and/or a method of the interface.
						vm.ConcreteTypeInfo[k.T].IsStruct()) { // Or hand it off to the Set method to return an error.
						vm.Mem[args[0]] = vm.makeError("vm/map/key", args[2], k, vm.DescribeType(k.T, LITERAL, 0))
						break Switch
					}
					result = result.Set(k, v)
				}
				vm.Mem[args[0]] = values.Value{values.MAP, result}
			case Mkpr: // Make pair (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.PAIR, []values.Value{vm.Mem[args[1]], vm.Mem[args[2]]}}
			case Mkst: // Make set (dst mem tok)
				// Constructs a map from a tuple value v#1, using token n#2 to create an error if the
				// elements of the tuple have the wrong type, e.g. there's an unhashable value.
				result := values.Set{}
				for _, v := range vm.Mem[args[1]].V.([]values.Value) {
					// TODO --- whether a type can be put in a set should be extractable from its concrete type information.
					result = result.Add(v)
				}
				vm.Mem[args[0]] = values.Value{values.SET, result}
			case MkSn: // Make snippet (dst sfc)
				// Here n#1 is a snippet factory analogous to a lambda factory.
				sFac := vm.SnippetFactories[args[1]]
				vals := make([]values.Value, len(sFac.Bindle.ValueLocs))
				for i, v := range sFac.Bindle.ValueLocs {
					vals[i] = vm.Mem[v]
				}
				vm.Mem[args[0]] = values.Value{values.SNIPPET, values.Snippet{vals, sFac.Bindle}}
			case Mlfi: // Multiply a float and an integer (dst mem mem)
				vm.Mem[args[0]] = values.Value{values.FLOAT, vm.Mem[args[1]].V.(float64) * float64(vm.Mem[args[2]].V.(int))}
			case Modi: // Modulus of integers (dst mem mem tok)
				divisor := vm.Mem[args[2]].V.(int)
				if divisor == 0 {
					vm.Mem[args[0]] = vm.makeError("vm/mod/zero", args[3])
				} else {
					vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(int) % vm.Mem[args[2]].V.(int)}
				}
			case Mpar: // Make parameterized type (dst ptp tok tup)
				// Operands :
				//     n#1 : the number of the parameterized type constructor.
				//     n#2 : number of a token for throwing an error if the value can't be constructed.
				//     n#3 : a tuple of arguments to pass to the constructor.
				vals := []values.Value{}
				for _, loc := range args[3:] {
					vals = append(vals, vm.Mem[loc])
				}
				entry, ok := vm.ParameterizedTypeInfo[args[1]].Get(values.Value{values.TUPLE, vals})
				if !ok {
					argsAsAny := []any{}
					for _, v := range vals {
						argsAsAny = append(argsAsAny, v)
					}
					vm.Mem[args[0]] = vm.makeError("vm/param/exist", args[2], argsAsAny...)
				} else {
					vm.Mem[args[0]] = entry
				}
			case Mulf: // Multiply floats (dst mem mem)
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(float64) * vm.Mem[args[2]].V.(float64)}
			case Muli: // Multiply ints (dst mem mem)
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(int) * vm.Mem[args[2]].V.(int)}
			case Negf: // Negate flaot (dst mem)
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, -vm.Mem[args[1]].V.(float64)}
			case Negi: // Negate int (dst mem)
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, -vm.Mem[args[1]].V.(int)}
			case Notb: // Binary not (dst mem)
				vm.Mem[args[0]] = values.Value{values.BOOL, !vm.Mem[args[1]].V.(bool)}
			case Outp: // Post to output (mem)
				vm.OutHandle.Out(vm.Mem[args[0]])
				vm.PostHappened = true
			case Outt: // Post to terminal (mem)
				if vm.Mem[vm.UsefulValues.OutputAs].V.(int) == 0 {
					fmt.Println(vm.Literal(vm.Mem[args[0]], 0))
				} else {
					fmt.Println(vm.DefaultDescription(vm.Mem[args[0]]))
				}
			case Psql: // Post to SQL (dst mem mem tok)
				// Here v#1 is the SQL accessor object and v#2 is a snippet. We return either `OK` or an 
				// error created using the token n#3
				sqlObj := vm.Mem[args[1]].V.(*sql.DB)
				snippet := vm.Mem[args[2]].V.(values.Snippet).Data
				buf := strings.Builder{}
				vals := []values.Value{}
				for i, v := range snippet {
					if i%2 == 0 {
						buf.WriteString(v.V.(string))
					} else {
						switch v.T {
						case values.TYPE: // We're turning a Pipefish type into the signature of a SQL table.
							if v.V.(values.AbstractType).Len() != 1 {
								vm.Mem[args[0]] = vm.makeError("vm/sql/abstract/d", args[3], vm.DescribeAbstractType(v.V.(values.AbstractType), LITERAL, 0))
								break Switch
							}
							cType := v.V.(values.AbstractType).Types[0]
							sqlSig, err := vm.getTableSigFromStructType(cType, args[3])
							if err.T == values.ERROR {
								vm.Mem[args[0]] = err
								break Switch
							}
							buf.WriteString(sqlSig)
						case values.TUPLE:
							valsToAdd := v.V.([]values.Value)
							sep := ""
							for i := len(vals) + 1; i < len(vals)+1+len(valsToAdd); i++ {
								buf.WriteString(sep)
								buf.WriteString("$")
								buf.WriteString(strconv.Itoa(i))
								sep = ", "
							}
							vals = append(vals, valsToAdd...)
						default:
							vals = append(vals, v)
							buf.WriteString("$")
							buf.WriteString(strconv.Itoa(len(vals)))
						}
					}
				}
				vm.Mem[args[0]] = vm.evalPostSQL(sqlObj, buf.String(), vals, args[3])
			case Qabt: // Test abstract type  (mem tup loc)
				// Jumps to the location n#2 if the type of v#0 is not in the tuple of type numbers in #1.
				for _, t := range args[1 : len(args)-1] {
					if vm.Mem[args[0]].T == values.ValueType(t) {
						loc = loc + 1
						continue loop
					}
				}
				loc = args[len(args)-1]
				continue
			case Qfls: // Test for false (mem loc)
				// This jumps to the location number n#1 if v#0 is not false.
				if vm.Mem[args[0]].V.(bool) {
					loc = args[1]
				} else {
					loc = loc + 1
				}
				continue
			case Qitr: // Test for end of iterator (mem loc)
				// This jumps to location number n#1 if the iterator v#0 hasn't finished iterating.
				if vm.Mem[args[0]].V.(Iterator).Unfinished() {
					loc = args[1]
				} else {
					loc = loc + 1
				}
				continue
			case QleT: // Test length of tuple <= n (mem num loc)
				// Jumps to location number n#2 if the length of the tuple value v#0 isn't less than or equal to n#1.
				if vm.Mem[args[0]].T == values.TUPLE && len(vm.Mem[args[0]].V.([]values.Value)) <= int(args[1]) {
					loc = loc + 1
				} else {
					loc = args[2]
				}
				continue
			case QlnT: // Test length of tuple < n (mem num loc)
				// Jumps to location number n#2 if the length of the tuple value v#0 isn't less than or equal to n#1.
				if len(vm.Mem[args[0]].V.([]values.Value)) == int(args[1]) {
					loc = loc + 1
				} else {
					loc = args[2]
				}
				continue
			case Qlog: // Jumps to location number n#0 if logging is turned off (loc)
				if vm.logging {
					loc = loc + 1
				} else {
					loc = args[0]
				}
				continue
			case Qnab: // Test not in abstract type  (mem tup loc)
				// Jumps to the location n#2 if the type of v#0 is in the tuple of type numbers in #1.
				for _, t := range args[1 : len(args)-1] {
					if vm.Mem[args[0]].T == values.ValueType(t) {
						loc = args[len(args)-1]
						continue loop
					}
				}
				loc = loc + 1
				continue
			case Qntp: // Test not of type  (mem typ loc)
				// Jumps to the location n#2 if the type of v#0 is not the type numbers in n#1.
				if vm.Mem[args[0]].T != values.ValueType(args[1]) {
					loc = loc + 1
				} else {
					loc = args[2]
				}
				continue
			case Qsat: // Test not `UNSAT` (mem loc)
				// Jumps to location n#1 if v#0 is an unsatisfied conditional.
				if vm.Mem[args[0]].T != values.UNSATISFIED_CONDITIONAL {
					loc = loc + 1
				} else {
					loc = args[1]
				}
				continue
			case Qsnq: // Test singleton (mem loc)
				// Jumps to location n#1 if v#2 is a tuple.
				if vm.Mem[args[0]].T >= values.NULL {
					loc = loc + 1
				} else {
					loc = args[1]
				}
				continue
			case Qtpt: // Test tuple types (mem num tup loc)
				// Jumps to location n#3 if the first n#1 elements of the tuple value v#0 don't have types corresponding
				// to the type numbers in n#2.
				vals := vm.Mem[args[0]].V.([]values.Value)
				slice := []values.Value{}
				if int(args[1]) <= len(vals) {
					slice = vals[args[1]:]
				}
				for _, v := range slice {
					var found bool
					for _, t := range args[2 : len(args)-1] {
						if v.T == values.ValueType(t) {
							found = true
							break
						}
					}
					if !found {
						loc = args[len(args)-1]
						continue loop
					}
				}
				loc = loc + 1
				continue
			case Qtru: // Test true (mem loc)
				// Jumps to location n#1 if v#0 isn't true
				if vm.Mem[args[0]].V.(bool) {
					loc = loc + 1
				} else {
					loc = args[1]
				}
				continue
			case Qtyp: // Test type membership (mem typ loc)
				// Jumps to location #2 if v#0 doesn't have type number n#1
				if vm.Mem[args[0]].T == values.ValueType(args[1]) {
					loc = loc + 1
				} else {
					loc = args[2]
				}
				continue
			case Ret: // Return ()
				// If the height of the return stack is strictly greater than it was when the vm's `.Run` 
				// method was called, then we pop the top off the return stack and jump to that location.
				// Otherwise we've finished exacuting `.Run` and can return.
				if len(vm.callstack) == stackHeight { // This is so that we can call "Run" when we have things on the stack and it will bottom out at the appropriate time.
					break loop
				}
				loc = vm.callstack[len(vm.callstack)-1]
				vm.callstack = vm.callstack[0 : len(vm.callstack)-1]
			case Rpop: // Pop recursion data ()
				rData := vm.recursionStack[len(vm.recursionStack)-1]
				vm.recursionStack = vm.recursionStack[:len(vm.recursionStack)-1]
				copy(vm.Mem[rData.loc:int(rData.loc)+len(rData.mems)], rData.mems)
			case Rpsh: // Push recursion data (num num)
				lowLoc := args[0]
				highLoc := args[1]
				memToSave := make([]values.Value, highLoc-lowLoc)
				copy(memToSave, vm.Mem[lowLoc:highLoc])
				vm.recursionStack = append(vm.recursionStack, recursionData{memToSave, lowLoc})
			case SliL: // Slice of list (dst mem mem tok)
				// v#1 is a list, v#2 is a pair; token n#3 is used to create errors for e.g. when the pair
				// is out of bounds.
				vec := vm.Mem[args[1]].V.(vector.Vector)
				ix := vm.Mem[args[2]].V.([]values.Value)
				if ix[0].T != values.INT {
					vm.Mem[args[0]] = vm.makeError("vm/slice/list/a", args[3], vm.DescribeType(ix[0].T, LITERAL, 0), args[1], args[2])
					break Switch
				}
				if ix[1].T != values.INT {
					vm.Mem[args[0]] = vm.makeError("vm/slice/list/b", args[3], vm.DescribeType(ix[1].T, LITERAL, 0), args[1], args[2])
					break Switch
				}
				if ix[0].V.(int) < 0 {
					vm.Mem[args[0]] = vm.makeError("vm/slice/list/c", args[3], vec, ix)
					break Switch
				}
				if ix[1].V.(int) < ix[0].V.(int) {
					vm.Mem[args[0]] = vm.makeError("vm/slice/list/d", args[3], ix[0].V.(int), ix[1].V.(int), args[1], args[2])
					break Switch
				}
				if vec.Len() < ix[1].V.(int) {
					vm.Mem[args[0]] = vm.makeError("vm/slice/list/e", args[3], ix[1].V.(int), vec.Len(), args[1], args[2])
					break Switch
				}
				vm.Mem[args[0]] = values.Value{values.LIST, vec.SubVector(ix[0].V.(int), ix[1].V.(int))}
			case Slis: // Slice of string (dst mem mem tok)
				// v#1 is a string, v#2 is a pair; token n#3 is used to create errors for e.g. when the pair
				// is out of bounds.
				str := vm.Mem[args[1]].V.(string)
				ix := vm.Mem[args[2]].V.([]values.Value)
				if ix[0].T != values.INT {
					vm.Mem[args[0]] = vm.makeError("vm/slice/string/a", args[3], vm.DescribeType(ix[0].T, LITERAL, 0), args[1], args[2])
					break Switch
				}
				if ix[1].T != values.INT {
					vm.Mem[args[0]] = vm.makeError("vm/slice/string/b", args[3], vm.DescribeType(ix[1].T, LITERAL, 0), args[1], args[2])
					break Switch
				}
				if ix[0].V.(int) < 0 {
					vm.Mem[args[0]] = vm.makeError("vm/slice/string/c", args[3], args[1], args[2])
					break Switch
				}
				if ix[1].V.(int) < ix[0].V.(int) {
					vm.Mem[args[0]] = vm.makeError("vm/slice/string/d", args[3], ix[0].V.(int), ix[1].V.(int), args[1], args[2])
					break Switch
				}
				if len(str) < ix[1].V.(int) {
					vm.Mem[args[0]] = vm.makeError("vm/slice/string/e", args[3], ix[1].V.(int), len(str), args[1], args[2])
					break Switch
				}
				vm.Mem[args[0]] = values.Value{values.STRING, str[ix[0].V.(int):ix[1].V.(int)]}
			case SliT: // Slice of tuple (dst mem mem tok)
				// v#1 is a tuple, v#2 is a pair; token n#3 is used to create errors for e.g. when the pair
				// is out of bounds.
				tup := vm.Mem[args[1]].V.([]values.Value)
				ix := vm.Mem[args[2]].V.([]values.Value)
				if ix[0].T != values.INT {
					vm.Mem[args[0]] = vm.makeError("vm/slice/tuple/a", args[3], vm.DescribeType(ix[0].T, LITERAL, 0), args[1], args[2])
					break Switch
				}
				if ix[1].T != values.INT {
					vm.Mem[args[0]] = vm.makeError("vm/slice/tuple/b", args[3], vm.DescribeType(ix[1].T, LITERAL, 0), args[1], args[2])
					break Switch
				}
				if ix[0].V.(int) < 0 {
					vm.Mem[args[0]] = vm.makeError("vm/slice/tuple/c", args[3], ix[0].V.(int), ix[1].V.(int), args[1], args[2])
					break Switch
				}
				if ix[1].V.(int) < ix[0].V.(int) {
					vm.Mem[args[0]] = vm.makeError("vm/slice/tuple/d", args[3], ix[0].V.(int), ix[1].V.(int), args[1], args[2])
					break Switch
				}
				if len(tup) < ix[1].V.(int) {
					vm.Mem[args[0]] = vm.makeError("vm/slice/tuple/e", args[3], ix[1].V.(int), len(tup), args[1], args[2])
					break Switch
				}
				vm.Mem[args[0]] = values.Value{values.TUPLE, tup[ix[0].V.(int):ix[1].V.(int)]}
			case SlTn: // Hard slice tuple (dst mem num)
				// Returns the tuple consisting of the elements from n#1 to the end of the tuple value v#1.
				vm.Mem[args[0]] = values.Value{values.TUPLE, (vm.Mem[args[1]].V.([]values.Value))[args[2]:]}
			case StrP: // Make parameterized struct. (dst tok tup)
				// Constructs a parameterized struct of type n#1 from the values in the memory locations given in #2.
				// An error will be constructed from token number n#1 if the struct can't be constructed.
				typeNo := vm.Mem[args[2]].V.(values.AbstractType).Types[0]
				fields := make([]values.Value, 0, len(args)-3)
				for _, loc := range args[3:] {
					fields = append(fields, vm.Mem[loc])
				}
				if typeCheck := vm.ConcreteTypeInfo[typeNo].(StructType).Validation; typeCheck != nil {
					vm.Mem[typeCheck.TokNumberLoc] = values.Value{values.INT, int(args[1])}
					for i, loc := range args[3:] {
						vm.Mem[typeCheck.InLoc+uint32(i)] = vm.Mem[loc]
					}
					vm.Mem[typeCheck.ResultLoc] = values.Value{typeNo, fields}
					vm.run(typeCheck.CallAddress, ctx, cancel)
					vm.Mem[args[0]] = vm.Mem[typeCheck.ResultLoc]
				} else {
					vm.Mem[args[0]] = values.Value{typeNo, fields}
				}
			case Strc: // Make struct (dst typ tup)
				// Constructs a struct of type n#1 from the values in the memory locations given in #2.
				fields := make([]values.Value, 0, len(args)-2)
				for _, loc := range args[2:] {
					fields = append(fields, vm.Mem[loc])
				}
				vm.Mem[args[0]] = values.Value{values.ValueType(args[1]), fields}
			case Strx: // String of value (dst mem)
				vm.Mem[args[0]] = values.Value{values.STRING, vm.DefaultDescription(vm.Mem[args[1]])}
			case Subf: // Subtract floats (dst mem mem)
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(float64) - vm.Mem[args[2]].V.(float64)}
			case Subi: // Subtract integers (dst mem mem)
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(int) - vm.Mem[args[2]].V.(int)}
			case SubS: // Subtract sets (dst mem mem)
				result := vm.Mem[args[1]].V.(values.Set).Subtract(vm.Mem[args[2]].V.(values.Set))
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, result}
			case Test: // Run tests (dst num)
				// This runs all the tests for a module, with the number being the compiler number.
				vm.Mem[args[0]] = values.Value{values.SUCCESSFUL_VALUE, nil}
				for _, test := range vm.Tests[args[1]] {
					vm.run(test.CallTo, ctx, cancel)
					if vm.Mem[test.Return].T == values.ERROR {
						vm.Mem[args[0]] = vm.Mem[test.Return]
						break
					}
				}
			case Thnk: // Initialize thunk (dst mem loc)
				// This will set m#0 to be a value of type THUNK with a payload of n#1 and n#2. The first of these
				// says where the result of unthinking the thunk will end up; the second says where the VM will have 
				// to jsr to to unthunk it.
				vm.Mem[args[0]] = values.Value{values.THUNK, values.Thunk{args[1], args[2]}}
			case Tinf: // Get info for type (dst mem)
				// This dumps the type info for the type into a list returned in m#0. This is done under the hood,
				// the user never sees the raw list.
				result := vector.Empty
				ty := vm.Mem[args[1]].V.(values.AbstractType)
				conc := (ty.Len() == 1)
				name := vm.DescribeAbstractType(ty, LITERAL, 0)
				operator := ""
				ix := strings.IndexRune(name, '{')
				if ix == -1 {
					operator = name
				} else {
					operator = name[:ix]
				}
				result = result.Conj(values.Value{values.STRING, name})
				types := values.Set{}
				for _, v := range ty.Types {
					concType := values.AbstractType{[]values.ValueType{v}}
					types = types.Add(values.Value{values.TYPE, concType})
				}
				result = result.Conj(values.Value{values.SET, types})
				result = result.Conj(values.Value{values.BOOL, conc &&
					!(vm.ConcreteTypeInfo[ty.Types[0]].IsClone() ||
						vm.ConcreteTypeInfo[ty.Types[0]].IsEnum() ||
						vm.ConcreteTypeInfo[ty.Types[0]].IsStruct())})
				result = result.Conj(values.Value{values.BOOL, conc && vm.ConcreteTypeInfo[ty.Types[0]].IsClone()})
				result = result.Conj(values.Value{values.BOOL, conc && vm.ConcreteTypeInfo[ty.Types[0]].IsEnum()})
				result = result.Conj(values.Value{values.BOOL, conc && vm.ConcreteTypeInfo[ty.Types[0]].IsStruct()})
				if ct, ok := vm.ConcreteTypeInfo[ty.Types[0]].(CloneType); ok {
					result = result.Conj(values.Value{values.TYPE, values.AbT(ct.Parent)})
				} else {
					result = result.Conj(values.Value{values.NULL, nil})
				}
				if et, ok := vm.ConcreteTypeInfo[ty.Types[0]].(EnumType); ok {
					result = result.Conj(et.ElementValues)
				} else {
					result = result.Conj(values.Value{values.NULL, nil})
				}
				if st, ok := vm.ConcreteTypeInfo[ty.Types[0]].(StructType); ok {
					result = result.Conj(st.LabelValues)
					vec := vector.Empty
					for _, ty := range st.AbstractStructFields {
						vec = vec.Conj(values.Value{values.TYPE, ty})
					}
					result = result.Conj(values.Value{values.LIST, vec})
				} else {
					result = result.Conj(values.Value{values.NULL, nil})
					result = result.Conj(values.Value{values.NULL, nil})
				}
				result = result.Conj(values.Value{values.STRING, operator})
				pVals := vector.Empty
				pTypes := vector.Empty
				switch typeIs := vm.ConcreteTypeInfo[ty.Types[0]].(type) {
				case CloneType:
					for _, v := range typeIs.TypeArguments {
						pVals = pVals.Conj(v)
						pTypes = pTypes.Conj(values.Value{values.TYPE, values.AbT(v.T)})
					}
					result = result.Conj(values.Value{values.LIST, pVals})
					result = result.Conj(values.Value{values.LIST, pTypes})
				case StructType:
					for _, v := range typeIs.TypeArguments {
						pVals = pVals.Conj(v)
						pTypes = pTypes.Conj(values.Value{values.TYPE, values.AbT(v.T)})
					}
					result = result.Conj(values.Value{values.LIST, pVals})
					result = result.Conj(values.Value{values.LIST, pTypes})
				default:
					result = result.Conj(values.Value{values.NULL, nil})
					result = result.Conj(values.Value{values.NULL, nil})
				}
				vm.Mem[args[0]] = values.Value{values.LIST, result}
			case Tnst: // Nonstandard test (dst mem mem loc tok)
				// Converts a boolean false or true into an error or OK and puts the result in dst, jumping to the loc
				// if it's an error. The test is "nonstandard" in that the boolean we're testing isn't produced by the
				// built-in comparison operators.
				// Operands :
				//     m#1 : the address of the boolean value
				//     m#2 : the condition being tested, as a string
				//     #3  : the location to jump to
				//     n#4 : the index of the token to use if producing an error
				switch vm.Mem[args[1]].T {
				case values.BOOL :
					if vm.Mem[args[1]].V.(bool) {
						vm.Mem[args[0]] = values.Value{values.SUCCESSFUL_VALUE, nil}
						loc = loc + 1
					} else {
						vm.Mem[args[0]] = vm.makeError("vm/test/nstd", args[4], vm.Mem[args[2]].V.(string))
						loc = args[3]
					}
				case values.SUCCESSFUL_VALUE:
					vm.Mem[args[0]] = values.Value{values.SUCCESSFUL_VALUE, nil}
					loc = loc + 1
				case values.ERROR :					
					vm.Mem[args[0]] = vm.Mem[args[1]]
					loc = args[3]
				default :
					vm.Mem[args[0]] = vm.makeError("vm/test/bool.a", args[4], vm.Mem[args[2]].V.(string), args[1])
					loc = args[3]
				}
				continue
			case Tstd: // Standard test (dst mem mem mem mem loc tok)
				// Converts a boolean false or true into an error or OK and puts the result in dst, jumping to the loc
				// if it's an error. The test is "standard" in that the boolean we're testing is produced by the
				// built-in comparison operators.
				// Operands :
				//     m#1 : the address of the boolean value
				//     m#2 : the condition being tested, as a string
				//     m#3 : the address of the lhs of the comparion.
				//     m#4 : the address of the rhs of the comparison
				//     #5  : the location to jump to
				//     n#6 : the index of the token to use if producing an error
				switch vm.Mem[args[1]].T {
				case values.BOOL :
					if vm.Mem[args[1]].V.(bool) {
						vm.Mem[args[0]] = values.Value{values.SUCCESSFUL_VALUE, nil}
						loc = loc + 1
					} else {
						vm.Mem[args[0]] = vm.makeError("vm/test/std", args[6], vm.Mem[args[2]].V.(string), vm.Literal(vm.Mem[args[3]], 0), vm.Literal(vm.Mem[args[4]], 0))
						loc = args[5]
					}
				case values.ERROR :					
					vm.Mem[args[0]] = vm.Mem[args[1]]
					loc = args[5]
				default :
					vm.Mem[args[0]] = vm.makeError("vm/test/bool.b", args[6], vm.Mem[args[2]].V.(string), args[1])
					loc = args[5]
				}
				continue
			case Tplf: // First element of tuple (dst mem tok)
				// Returns the first element of a tuple, or an error created from token n#2 if the tuple is empty.
				tup := vm.Mem[args[1]].V.([]values.Value)
				if len(tup) == 0 {
					vm.Mem[args[0]] = vm.makeError("vm/tup/first", args[2])
					break Switch
				}
				vm.Mem[args[0]] = tup[0]
			case Trak: // Make tracking data (trk)
				// This constructs live tracking data saying what the compiler is doing now from the static tracking 
				// datum number n#0
				staticData := vm.Tracking[args[0]]
				newData := TrackingData{staticData.Flavor, staticData.Tok, staticData.LogToLoc, staticData.LogTimeLoc, make([]any, len(staticData.Args))}
				copy(newData.Args, staticData.Args) // This is because only things of type uint32 are meant to be replaced.
				for i, v := range newData.Args {
					if v, ok := v.(uint32); ok {
						newData.Args[i] = vm.Mem[v]
					}
				}
				vm.LiveTracking = append(vm.LiveTracking, newData)
			case TuLx: // Tuple of possible list (dst mem tok)
				// Splats the list if it is a list, otherwise returns an error constructed from token t#2.
				vector, ok := vm.Mem[args[1]].V.(vector.Vector)
				if !ok {
					vm.Mem[args[0]] = vm.makeError("vm/splat/type", args[2], args[1])
					break Switch
				}
				length := vector.Len()
				slice := make([]values.Value, length)
				for i := 0; i < length; i++ {
					element, _ := vector.Index(i)
					slice[i] = element.(values.Value)
				}
				vm.Mem[args[0]] = values.Value{values.TUPLE, slice}
			case TupL: // Tuple of list (dst mem)
				// That is, this implements the splat operator `L ...`.
				vector := vm.Mem[args[1]].V.(vector.Vector)
				length := vector.Len()
				slice := make([]values.Value, length)
				for i := 0; i < length; i++ {
					element, _ := vector.Index(i)
					slice[i] = element.(values.Value)
				}
				vm.Mem[args[0]] = values.Value{values.TUPLE, slice}
			case Typu: // Type union (dst mem mem)
				// That is, this implements `typeA/typeB`.
				lhs := vm.Mem[args[1]].V.(values.AbstractType)
				rhs := vm.Mem[args[2]].V.(values.AbstractType)
				vm.Mem[args[0]] = values.Value{values.TYPE, lhs.Union(rhs)}
			case Typx: // Type of value (dst mem)
				vm.Mem[args[0]] = values.Value{values.TYPE, values.AbstractType{[]values.ValueType{vm.Mem[args[1]].T}}}
			case Unsf: // Unsafe cast to parameterized clone type (dst mem mem tok)
				// Casts the value v#1 to the type v#2, where v#2 is (presumably) a parameterized clone type.
				// Token n#3 can be used to return an error if the conversion is impossible.
				abtype := vm.Mem[args[2]].V.(values.AbstractType).Types
				if len(abtype) != 1 {
					println("concrete.b")
					vm.Mem[args[0]] = vm.makeError("vm/cast/concrete.b", args[3])
					break Switch
				}
				typeNo := vm.Mem[args[2]].V.(values.AbstractType).Types[0]
				if info, ok := vm.ConcreteTypeInfo[typeNo].(CloneType); !ok {
					println("unsafe/clone")
					vm.Mem[args[0]] = vm.makeError("vm/unsafe/clone", args[3])
						break Switch
				} else {
					if info.Parent != vm.Mem[args[1]].T {
						println("cast/parent")
						vm.Mem[args[0]] = vm.makeError("vm/cast/parent", args[3])
						break Switch
					}
				}
				vm.Mem[args[0]] = values.Value{typeNo, vm.Mem[args[1]].V}
			case UntE: // Unthunk error (dst mem)
				// This takes the error v#1, converts all the arguments of the error of type uint32 to the values
				// in the corresponding memory locations, and returns it in m#0.
				oldErr := vm.Mem[args[1]].V.(*err.Error)
				newArgs := []any{}
				newVals := []values.Value{}
				for _, arg := range oldErr.Args {
					switch arg := arg.(type) {
					case uint32:
						newVals = append(newVals, vm.Mem[arg])
					case DescribeTypeOfValueAtLocation:
						newArgs = append(newArgs, vm.DescribeType(vm.Mem[arg].T, LITERAL, 0))
					default:
						newArgs = append(newArgs, arg)
					}
				}
				vm.Mem[args[0]] = values.Value{values.ERROR,
					&err.Error{oldErr.ErrorId, oldErr.Message, "", newArgs, newVals, []*token.Token{}, oldErr.Token}}
			case Untk: // Unthunk (dst)
				// This checks whether v#1 is of type THUNK. If it is, it `jsr`s to the code address contained in the thunk,
				// gets the evaluated result of the thunk, and puts it into m#0; otherwise it does nothing.
				if vm.Mem[args[0]].T == values.THUNK {
					resultLoc := vm.Mem[args[0]].V.(values.Thunk).MLoc
					codeAddr := vm.Mem[args[0]].V.(values.Thunk).CAddr
					vm.run(codeAddr, ctx, cancel)
					vm.Mem[args[0]] = vm.Mem[resultLoc]
				}
			case Uwrp: // Unwrap error (dst mem tok)
				// This turns something of type `error` into something of type `Error`, and ordinary struct defined in the
				// builtins. The token n#3 is used to return an error if we're trying to unwrap something that is not in
				// fact of type error
				if vm.Mem[args[1]].T == values.ERROR {
					wrappedErr := vm.Mem[args[1]].V.(*err.Error)
					errWithMessage := wrappedErr
					if wrappedErr.Message == "" {
						errWithMessage = err.CreateErr(wrappedErr.ErrorId, wrappedErr.Token, wrappedErr.Args...)
					}
					vm.Mem[args[0]] = values.Value{vm.UsefulTypes.UnwrappedError, []values.Value{{values.STRING, errWithMessage.ErrorId}, {values.STRING, errWithMessage.Message}}}
				} else {
					vm.Mem[args[0]] = vm.makeError("vm/unwrap", args[2], vm.DescribeType(vm.Mem[args[1]].T, LITERAL, 0))
				}
			case Vlid: // Valid (dst mem)
				// Returns `true` if v#1 is not of type error
				vm.Mem[args[0]] = values.Value{values.BOOL, vm.Mem[args[1]].T != values.ERROR}
			case WrHb: // Write to hub (mem mem)
				// A magical gizmo that lets services which are also hubs tell hub.go what to do. v#0 is a string saying
				// which hub action we want to take, and v#2 is a list containing parameters.
				vm.Mem[args[1]].V.(io.Writer).Write([]byte(vm.Mem[args[2]].V.(string)))
				vm.Mem[args[0]] = values.Value{values.SUCCESSFUL_VALUE, nil}
			case WthL: // List with (dst mem tok tup)
				// The `with` operator for lists. v#1 is a list, #2 is a tuple of pairs, and token n#2 is for constructing
				// an error if the pairs are wrong, e.g. if the key of a pair is outside the bounds of the list.
				result := values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(vector.Vector)}
				for it := vm.NewValueIterator(args[3:]); ; {
					pair, ok := it.get()
					if !ok {
						break
					}
					key := pair.V.([]values.Value)[0]
					val := pair.V.([]values.Value)[1]
					var keys []values.Value
					if key.T == values.LIST {
						vec := key.V.(vector.Vector)
						ln := vec.Len()
						if ln == 0 {
							vm.Mem[args[0]] = vm.makeError("vm/with/list/b", args[2])
							break Switch
						}
						keys = make([]values.Value, ln)
						for i := 0; i < ln; i++ {
							el, _ := vec.Index(i)
							keys[i] = el.(values.Value)
						}
					} else {
						keys = []values.Value{key}
					}
					result = vm.with(result, keys, val, args[2])
					if result.T == values.ERROR {
						break
					}
				}
				vm.Mem[args[0]] = result
			case WthM: // Map with (dst mem tok tup)
				// The `with` operator for maps. v#1 is a map, #2 is a tuple of pairs, and token n#2 is for constructing
				// an error if the pairs are wrong, e.g. if the key of a pair is unhashable.
				result := values.Value{vm.Mem[args[1]].T, vm.Mem[args[1]].V.(values.Map)}
				for it := vm.NewValueIterator(args[3:]); ; {
					pair, ok := it.get()
					if !ok {
						break
					}
					key := pair.V.([]values.Value)[0]
					val := pair.V.([]values.Value)[1]
					var keys []values.Value
					if key.T == values.LIST {
						vec := key.V.(vector.Vector)
						ln := vec.Len()
						if ln == 0 {
							vm.Mem[args[0]] = vm.makeError("vm/with/map/b", args[2])
							break Switch
						}
						keys = make([]values.Value, ln)
						for i := 0; i < ln; i++ {
							el, _ := vec.Index(i)
							keys[i] = el.(values.Value)
						}
					} else {
						keys = []values.Value{key}
					}
					result = vm.with(result, keys, val, args[2])
					if result.T == values.ERROR {
						break
					}
				}
				vm.Mem[args[0]] = result
			case WthT: // Long-form type constructor (dst mem tok tup)
				// v#1 is a type, token n#2 is for constructing an error, and the tup is or should be pairs of labels and values.
				typL := vm.Mem[args[1]].V.(values.AbstractType)
				if typL.Len() != 1 {
					vm.Mem[args[0]] = vm.makeError("vm/with/type/a", args[2], vm.DescribeAbstractType(typL, LITERAL, 0))
					break Switch
				}
				typ := typL.Types[0]
				if !vm.ConcreteTypeInfo[typ].IsStruct() {
					vm.Mem[args[0]] = vm.makeError("vm/with/type/b", args[2], vm.DescribeType(typ, LITERAL, 0))
					break Switch
				}
				typeInfo := vm.ConcreteTypeInfo[typ].(StructType)
				outVals := make([]values.Value, len(vm.ConcreteTypeInfo[typ].(StructType).LabelNumbers))
				for it := vm.NewValueIterator(args[3:]); ; {
					pair, ok := it.get()
					if !ok {
						break
					}
					key := pair.V.([]values.Value)[0]
					val := pair.V.([]values.Value)[1]
					if key.T != values.LABEL {
						vm.Mem[args[0]] = vm.makeError("vm/with/type/d", args[2], vm.DescribeType(pair.T, LITERAL, 0))
						break Switch
					}
					keyNumber := typeInfo.Resolve(key.V.(int))
					if keyNumber == -1 {
						vm.Mem[args[0]] = vm.makeError("vm/with/type/e", args[2], vm.DefaultDescription(key), vm.DescribeType(typ, LITERAL, 0))
						break Switch
					}
					outVals[keyNumber] = val
				}
				for i, v := range outVals {
					if v.T == values.UNDEFINED_TYPE { // As a special case, we don't need to specify that nullable things are `NULL`.
						if vm.ConcreteTypeInfo[typ].(StructType).AbstractStructFields[i].Contains(values.NULL) {
							outVals[i] = values.Value{values.NULL, nil}
						} else { // Otherwise, omitting a field is an error.
							labName := vm.Labels[vm.ConcreteTypeInfo[typ].(StructType).LabelNumbers[i]]
							vm.Mem[args[0]] = vm.makeError("vm/with/type/g", args[2], labName)
							break Switch
						}
					}
					if !vm.ConcreteTypeInfo[typ].(StructType).AbstractStructFields[i].Contains(outVals[i].T) {
						labName := vm.Labels[vm.ConcreteTypeInfo[typ].(StructType).LabelNumbers[i]]
						vm.Mem[args[0]] = vm.makeError("vm/with/type/h", args[2], vm.DescribeType(v.T, LITERAL, 0), labName, vm.DescribeType(typ, LITERAL, 0), vm.DescribeAbstractType(vm.ConcreteTypeInfo[typ].(StructType).AbstractStructFields[i], LITERAL, 0))
						break Switch
					}
				}
				// It may need validation.
				typecheck := vm.ConcreteTypeInfo[typ].(StructType).Validation
				if typecheck == nil {
					vm.Mem[args[0]] = values.Value{typ, outVals}
				} else {
					vm.Mem[typecheck.TokNumberLoc] = values.Value{values.INT, int(args[2])}
					for i, v := range outVals {
						vm.Mem[typecheck.InLoc+uint32(i)] = v
					}
					vm.Mem[typecheck.ResultLoc] = values.Value{typ, outVals}
					vm.run(typecheck.CallAddress, ctx, cancel)
					vm.Mem[args[0]] = vm.Mem[typecheck.ResultLoc]
				}
			case WthZ: // Struct with (dst mem tok tup)
				// The `with` operator for structs. v#1 is a struct, #2 is a tuple of pairs, and token n#2 is for constructing
				// an error if the pairs are wrong, e.g. if the key of a pair is not a field of the struct.
				typ := vm.Mem[args[1]].T
				outVals := make([]values.Value, len(vm.ConcreteTypeInfo[typ].(StructType).LabelNumbers))
				copy(outVals, vm.Mem[args[1]].V.([]values.Value))
				result := values.Value{typ, outVals}
				for it := vm.NewValueIterator(args[3:]); ; {
					pair, ok := it.get()
					if !ok {
						break
					}
					key := pair.V.([]values.Value)[0]
					val := pair.V.([]values.Value)[1]
					var keys []values.Value
					if key.T == values.LIST {
						vec := key.V.(vector.Vector)
						ln := vec.Len()
						if ln == 0 {
							vm.Mem[args[0]] = vm.makeError("vm/with/struct/b", args[2])
							break Switch
						}
						keys = make([]values.Value, ln)
						for i := 0; i < ln; i++ {
							el, _ := vec.Index(i)
							keys[i] = el.(values.Value)
						}
					} else {
						keys = []values.Value{key}
					}
					result = vm.with(result, keys, val, args[2])
					if result.T == values.ERROR {
						break
					}
				}
				vm.Mem[args[0]] = result
			case WtoM: // Map without (dst mem tok tup)
				// v#1 is a map, and #3 is a tuple of key values to be removed from it. Token n#2 is for constructing
				// an error if any of the values is unhashable unhashable
				mp := vm.Mem[args[1]].V.(values.Map)
				for it := vm.NewValueIterator(args[3:]); ; {
					key, ok := it.get()
					if !ok {
						break
					}
					if key.T < values.NULL || key.T == values.FUNC { // Check that the key is orderable.
						vm.Mem[args[0]] = vm.makeError("vm/without", args[2], vm.DescribeType(key.T, LITERAL, 0))
						break Switch
					}
					mp = (mp).Delete(key)
				}
				vm.Mem[args[0]] = values.Value{vm.Mem[args[1]].T, mp}
			case Yeet: // Yeet type parameters (dst mem)
				// If v#1 is of a parameterized type, then this assigns the parameters of this type to m#0 and to the
				// following memory addresses, one address for each parameter.
				typeInfo := vm.ConcreteTypeInfo[vm.Mem[args[1]].T]
				var typeArgs []values.Value
				switch typeInfo := typeInfo.(type) {
				case StructType:
					typeArgs = typeInfo.TypeArguments
				case CloneType:
					typeArgs = typeInfo.TypeArguments
				default:
					panic("Unhandled case.")
				}
				for i, v := range typeArgs {
					vm.Mem[args[0]+uint32(i)] = v
				}
			default:
				panic("Unhandled opcode '" + opInfo[vm.Code[loc].Opcode].opcode + "'")
			}
			loc++
		}
	}
}

// Implements equality-by-value. Assumes that the two values have already been verified to have the same type.
func (vm Vm) equals(v, w values.Value) bool {
	switch v.T {
	case values.BOOL:
		return v.V.(bool) == w.V.(bool)
	case values.FLOAT:
		return v.V.(float64) == w.V.(float64)
	case values.FUNC:
		return false
	case values.INT:
		return v.V.(int) == w.V.(int)
	case values.LABEL:
		return v.V.(int) == w.V.(int)
	case values.LIST:
		return vm.listsAreEqual(v, w)
	case values.MAP:
		return vm.mapsAreEqual(v, w)
	case values.NULL:
		return true
	case values.PAIR:
		return v.V.([]values.Value)[0].T == w.V.([]values.Value)[0].T &&
			v.V.([]values.Value)[1].T == w.V.([]values.Value)[1].T &&
			vm.equals(v.V.([]values.Value)[0], w.V.([]values.Value)[0]) &&
			vm.equals(v.V.([]values.Value)[1], w.V.([]values.Value)[1])
	case values.RUNE:
		return v.V.(rune) == w.V.(rune)
	case values.SET:
		return vm.setsAreEqual(v, w)
	case values.SNIPPET:
		vVals := v.V.(values.Snippet)
		wVals := w.V.(values.Snippet)
		if len(vVals.Data) != len(wVals.Data) {
			return false
		}
		for i, val := range vVals.Data {
			if val.T != wVals.Data[i].T {
				return false
			}
			if !vm.equals(val, wVals.Data[i]) {
				return false
			}
		}
		return true
	case values.STRING:
		return v.V.(string) == w.V.(string)
	case values.SUCCESSFUL_VALUE:
		return true
	case values.TUPLE:
		vVals := v.V.([]values.Value)
		wVals := w.V.([]values.Value)
		if len(vVals) != len(wVals) {
			return false
		}
		for i, val := range vVals {
			if val.T != wVals[i].T {
				return false
			}
			if !vm.equals(val, wVals[i]) {
				return false
			}
		}
		return true
	case values.TYPE:
		return v.V.(values.AbstractType).Equals(w.V.(values.AbstractType))
	}
	switch typeInfo := vm.ConcreteTypeInfo[v.T].(type) {
	case CloneType:
		switch typeInfo.Parent {
		case values.FLOAT:
			return v.V.(float64) == w.V.(float64)
		case values.INT:
			return v.V.(int) == w.V.(int)
		case values.LIST:
			return vm.listsAreEqual(v, w)
		case values.MAP:
			return vm.mapsAreEqual(v, w)
		case values.PAIR:
			return vm.equals(v.V.([]values.Value)[0], w.V.([]values.Value)[0]) &&
				vm.equals(v.V.([]values.Value)[1], w.V.([]values.Value)[1])
		case values.RUNE:
			return v.V.(rune) == w.V.(rune)
		case values.SET:
			return vm.setsAreEqual(v, w)
		case values.STRING:
			return v.V.(string) == w.V.(string)
		}
	case EnumType:
		return v.V.(int) == w.V.(int)
	case WrapperType:
		if vm.GoEquals == nil {
			return false
		}
		return vm.GoEquals(v.V, w.V)
	case StructType:
		for i, v := range v.V.([]values.Value) {
			if !vm.equals(v, w.V.([]values.Value)[i]) {
				return false
			}
		}
		return true
	}
	panic("Wut?")
}

func (vm *Vm) listsAreEqual(v, w values.Value) bool {
	K := v.V.(vector.Vector)
	L := w.V.(vector.Vector)
	lth := K.Len()
	if L.Len() != lth {
		return false
	}
	for i := 0; i < lth; i++ {
		kEl, _ := K.Index(i)
		lEl, _ := L.Index(i)
		if kEl.(values.Value).T != lEl.(values.Value).T {
			return false
		}
		if !vm.equals(kEl.(values.Value), lEl.(values.Value)) {
			return false
		}
	}
	return true
}

func (vm *Vm) mapsAreEqual(v, w values.Value) bool {
	mapV := v.V.(values.Map)
	mapW := w.V.(values.Map)
	if mapV.Len() != mapW.Len() {
		return false
	}
	sl := mapV.AsSlice()
	for _, pair := range sl {
		if val, ok := mapW.Get(pair.Key); !ok || !vm.equals(pair.Val, val) {
			return false
		}
	}
	return true
}

func (vm *Vm) setsAreEqual(v, w values.Value) bool {
	setV := v.V.(values.Set)
	setW := w.V.(values.Set)
	if setV.Len() != setW.Len() {
		return false
	}
	sl := setV.AsSlice()
	for _, el := range sl {
		if !setW.Contains(el) {
			return false
		}
	}
	return true
}

// Implements `with`, which needs to be done separately because it may be recursive.
func (vm *Vm) with(container values.Value, keys []values.Value, val values.Value, errTok uint32) values.Value {
	key := keys[0]
	parentType := container.T
	info := vm.ConcreteTypeInfo[container.T]
	if cloneInfo, ok := info.(CloneType); ok {
		parentType = cloneInfo.Parent
	}
	switch parentType {
	case values.LIST:
		vec := container.V.(vector.Vector)
		if key.T != values.INT {
			return vm.makeError("vm/with/a", errTok, vm.DescribeType(key.T, LITERAL, 0))
		}
		keyNumber := key.V.(int)
		if keyNumber < 0 || keyNumber >= vec.Len() {
			return vm.makeError("vm/with/b", errTok, key.V.(int), vec.Len())
		}
		if len(keys) == 1 {
			container.V = vec.Assoc(keyNumber, val)
			return container
		}
		el, _ := vec.Index(keyNumber)
		container.V = vec.Assoc(keyNumber, vm.with(el.(values.Value), keys[1:], val, errTok))
		return container
	case values.MAP:
		mp := container.V.(values.Map)
		if ((key.T < values.NULL) || (key.T >= values.FUNC && key.T < values.LABEL)) && !vm.ConcreteTypeInfo[key.T].IsEnum() { // Check that the key is orderable.
			return vm.makeError("vm/with/c", errTok, vm.DescribeType(key.T, LITERAL, 0))
		}
		if len(keys) == 1 {
			mp = mp.Set(key, val)
			return values.Value{container.T, mp}
		}
		el, _ := mp.Get(key)
		mp = mp.Set(key, vm.with(el, keys[1:], val, errTok))
		return values.Value{container.T, mp}
	default: // It's a struct.
		fields := make([]values.Value, len(container.V.([]values.Value)))
		clone := values.Value{container.T, fields}
		copy(fields, container.V.([]values.Value))
		typeInfo := vm.ConcreteTypeInfo[container.T].(StructType)
		if key.T != values.LABEL {
			return vm.makeError("vm/with/d", errTok, vm.DescribeType(key.T, LITERAL, 0))
		}
		fieldNumber := typeInfo.Resolve(key.V.(int))
		if fieldNumber == -1 {
			return vm.makeError("vm/with/e", errTok, vm.DefaultDescription(key), vm.DescribeType(container.T, LITERAL, 0))
		}
		if len(keys) > 1 {
			val = vm.with(fields[fieldNumber], keys[1:], val, errTok)
		}
		if !vm.ConcreteTypeInfo[container.T].(StructType).AbstractStructFields[fieldNumber].Contains(val.T) {
			labName := vm.Labels[key.V.(int)]
			return vm.makeError("vm/with/f", errTok, vm.DescribeType(val.T, LITERAL, 0), labName, vm.DescribeType(container.T, LITERAL, 0), vm.DescribeAbstractType(vm.ConcreteTypeInfo[container.T].(StructType).AbstractStructFields[fieldNumber], LITERAL, 0))
		}
		fields[fieldNumber] = val
		return clone
	}
}

// Produces a Value of the internal type ITERATOR for use in implementing `for` loops.
func (vm *Vm) NewIterator(container values.Value, keysOnly bool, tokLoc uint32) values.Value {
	ty := container.T
	if cloneInfo, ok := vm.ConcreteTypeInfo[ty].(CloneType); ok {
		ty = cloneInfo.Parent
	}
	switch ty {
	case values.INT:
		return values.Value{values.ITERATOR, &IncIterator{StartVal: 0, MaxVal: container.V.(int), Val: 0}}
	case values.LIST:
		if keysOnly {
			return values.Value{values.ITERATOR, &KeyIncIterator{Max: container.V.(vector.Vector).Len()}}
		} else {
			return values.Value{values.ITERATOR, &ListIterator{VecIt: container.V.(vector.Vector).Iterator()}}
		}
	case values.MAP:
		mapAsSlice := container.V.(values.Map).AsSlice()
		return values.Value{values.ITERATOR, &MapIterator{KVPairs: mapAsSlice, Len: len(mapAsSlice)}}
	case values.PAIR:
		pair := container.V.([]values.Value)
		left := pair[0]
		right := pair[1]
		if left.T != values.INT || right.T != values.INT {
			return vm.makeError("vm/for/pair", tokLoc)
		}
		leftV := left.V.(int)
		rightV := right.V.(int)
		if leftV <= rightV {
			return values.Value{values.ITERATOR, &IncIterator{StartVal: leftV, MaxVal: rightV, Val: leftV}}
		} else {
			return values.Value{values.ITERATOR, &DecIterator{MinVal: rightV, StartVal: leftV - 1, Val: leftV - 1}}
		}
	case values.SET:
		setAsSlice := container.V.(values.Set).AsSlice()
		return values.Value{values.ITERATOR, &SetIterator{Elements: setAsSlice, Len: len(setAsSlice)}}
	case values.SNIPPET:
		if keysOnly {
			return values.Value{values.ITERATOR, &KeyIncIterator{Max: len(container.V.(values.Snippet).Data)}}
		} else {
			return values.Value{values.ITERATOR, &TupleIterator{Elements: container.V.(values.Snippet).Data, Len: len(container.V.(values.Snippet).Data)}}
		}
	case values.STRING:
		return values.Value{values.ITERATOR, &StringIterator{Str: container.V.(string)}}
	case values.TUPLE:
		tupleElements := container.V.([]values.Value)
		if keysOnly {
			return values.Value{values.ITERATOR, &KeyIncIterator{Max: len(tupleElements)}}
		} else {
			return values.Value{values.ITERATOR, &TupleIterator{Elements: tupleElements, Len: len(tupleElements)}}
		}
	case values.TYPE:
		abTyp := container.V.(values.AbstractType)
		if len(abTyp.Types) != 1 {
			return vm.makeError("vm/for/type/a", tokLoc)
		}
		typ := abTyp.Types[0]
		if !vm.ConcreteTypeInfo[typ].IsEnum() {
			return vm.makeError("vm/for/type/b", tokLoc)
		}
		if keysOnly {
			return values.Value{values.ITERATOR, &KeyIncIterator{Max: len(vm.ConcreteTypeInfo[typ].(EnumType).ElementNames)}}
		} else {
			return values.Value{values.ITERATOR, &EnumIterator{Type: typ, Max: len(vm.ConcreteTypeInfo[typ].(EnumType).ElementNames)}}
		}
	}
	if typeInfo, ok := vm.ConcreteTypeInfo[ty].(StructType); ok {
		return values.Value{values.ITERATOR, &StructIterator{Labels: typeInfo.LabelNumbers, Values: container.V.([]values.Value)}}
	}
	return vm.makeError("vm/for/type/c", tokLoc)
}

// Constants for describing the syntax of functions.
const (
	PREFIX uint32 = iota
	INFIX
	SUFFIX
	UNFIX
)

// This takes the vm and a list of memory locations as arguments, and returns an iterator which
// which will return one value at a time, automatically decomposing any tuples.
//
// We do this because as things like `WthL` don't pass through `CalT`, they don't autosplat tuples,
// and this is a way of doing it for us.
//
// We don't just use something from `iter` because we want some extra logic.
//
// TODO --- normally we can detect at compile time if there are going to be any tuples involved
// and compile to simpler logic if it isn't. This would involves making more opcodes, or a flag,
// or something.
type ValueIterator struct {
	vm    *Vm
	locs  []uint32
	locNo int
	pos   int
}

func (vit *ValueIterator) get() (values.Value, bool) {
	for {
		if vit.locNo >= len(vit.locs) {
			return values.Value{}, false
		}
		val := vit.vm.Mem[vit.locs[vit.locNo]]
		if val.T == values.TUPLE {
			vals := val.V.([]values.Value)
			if vit.pos >= len(vals) {
				vit.pos = 0
				vit.locNo++
				continue
			}
			vit.pos++
			return vals[vit.pos-1], true
		}
		vit.locNo++
		return val, true
	}
}

func (vm *Vm) NewValueIterator(locs []uint32) *ValueIterator {
	return &ValueIterator{vm: vm, locs: locs}
}
















































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































































