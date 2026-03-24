package values

type ValueType uint32

const ( // Cross-reference with typeNames in BlankVm()

	// The types from UNDEFINED VALUE to BREAK inclusive are internal types which should never actually be seen by the user.
	// In some cases, e.g. CREATED_LOCAL_CONSTANT, they are also not instantiated: they are there to
	// return in a TypeScheme object when the compiled code doesn't create a value.

	UNDEFINED_TYPE          ValueType = iota // For debugging purposes, it is useful to have the zero value be something it should never actually be.
	TUPLE_DATA                               // Passes data to `CalT` about functions with tuples etc, to help it caputre them properly.`
	THUNK                                    // V is a ThunkValue which contains the address to call to evaluate the thunk and the memory location where the result ends up.
	CREATED_THUNK_OR_CONST                   // Returned by the compiler in the TypeScheme when we compile a thunk.
	BLING                                    // Values representing e.g. the `troz` in `foo (x) troz (y)`.
	UNSATISFIED_CONDITIONAL                  // An unsatisfied conditional, i.e. what <condition> : <expression> returns if <condition> isn't true.
	REF                                      // A reference variable. This is always dereferenced when used, so the type is invisible.
	ITERATOR                                 // V is an Iterator interface as defined in iterator.go in this folder.

	// And now we have types visible to the user.

	SUCCESSFUL_VALUE   // V : nil
	TUPLE              // V : []values.Value
	ERROR              // V : *err.Error
	NULL               // V : nil
	INT                // V : int
	BOOL               // V : bool
	STRING             // V : string
	RUNE               // V : string
	FLOAT              // V : float
	TYPE               // V : abstractType
	FUNC               // V : vm.Lambda
	PAIR               // V : []values.Value // TODO --- this should be [2]values.Value just for neatness.
	LIST               // V : vector.Vector
	MAP                // V : values.Map
	SET                // V : values.Set
	LABEL              // V : int
	SNIPPET            // V : Snippet struct{Data []values.Value, Bindle *SnippetBindle}
	FIRST_DEFINED_TYPE // I.e the first of the enums.
)

const DUMMY = 4294967295

type Value struct {
	T ValueType
	V any
}

type Snippet struct {
	Data   []Value
	Bindle *SnippetBindle
}

// A grouping of all the things a snippet from a given snippet factory have in common.
type SnippetBindle struct {
	CodeLoc   uint32   // Where to find the code to compute the object string and the values.
	ValueLocs []uint32 // The locations where we put the computed values to inject into SQL or HTML snippets.
}

// To implement the set and hash structures.
// If the type of the value is not comparable, we return that values so we can use it to
// make an error as required. We return OK for success (this is in fact comparable.)
func (v Value) compare(w Value) bool {
	//It doesn't really matter which order these things are in, so long as there is one.
	// TODO --- these next few lines will, alas, let us compare things that can't be compared, we need a filter, possibly at the VM end.
	if v.T < w.T {
		return true
	}
	if w.T < v.T {
		return false
	}
	// At this point since the two values must have the same internal representation we can
	// compare them without knowing what the type is, which is a good thing because we don't
	// have access to the ConcreteTypeInfo in the VM.
	switch lhs := v.V.(type) {
	case bool:
		return (!lhs) && w.V.(bool)
	case float64:
		return lhs < w.V.(float64)
	case int:
		return lhs < w.V.(int)
	case nil:
		return false
	case rune:
		return lhs < w.V.(rune)
	case string:
		return lhs < w.V.(string)
	case []Value:
		if len(lhs) == len(w.V.([]Value)) {
			for i, vEl := range lhs {
				if vEl.compare(w.V.([]Value)[i]) {
					return true
				}
			}
			return false
		}
		return len(lhs) < len(w.V.([]Value))
	case AbstractType:
		rhs := w.V.(AbstractType)
		if len(lhs.Types) == len(rhs.Types) {
			for i, ty := range lhs.Types {
				if ty < rhs.Types[i] {
					return true
				}
				if ty > rhs.Types[i] {
					return false
				}
			}
			return false
		} else {
			return len(lhs.Types) < len(rhs.Types)
		}
	}
	panic("This is why you need to implement guards on the types.")
}

// Cross-reference with CONSTANTS in vm.go.
var (
	UNDEF = Value{UNDEFINED_TYPE, nil}
	FALSE = Value{BOOL, false}
	TRUE  = Value{BOOL, true}
	U_OBJ = Value{T: UNSATISFIED_CONDITIONAL}
	ZERO  = Value{INT, 0}
	ONE   = Value{INT, 1}
	BLNG  = Value{BLING, "bling"}
	OK    = Value{SUCCESSFUL_VALUE, nil}
	EMPTY = Value{TUPLE, []Value{}}
)

const (
	C_UNDEF = iota
	C_FALSE
	C_TRUE
	C_UNSAT
	C_ZERO
	C_ONE
	C_BLING
	C_OK
	C_EMPTY_TUPLE
)

