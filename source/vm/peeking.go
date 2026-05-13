package vm

import (
	"strings"

	"github.com/tim-hardcastle/pipefish/source/token"
)

// This contains temporary or permanent code for peeking at the operations of the VM.

// This will just be a whitespace-separated string like "foo bar !qux", where ! indicates a flag
// to be turned off.
func (vm *Vm) SetPeeks(s string) {
	peekList := strings.Fields(s)
	peeks := map[string]bool{}
	for _, item := range peekList {
		if item[0] == '!' {
			peeks[item[1:]] = false
		} else {
			peeks[item] = true
		}
	}
	vm.PeekStack = append(vm.PeekStack, peeks)
}

func (vm *Vm) SetPeeksFromTokens(toks []token.Token) {
	peeks := map[string]bool{}
	negated := false
	for _, item := range toks {
		if item.Literal == "!" {
			negated = true
			continue
		} 
		peeks[item.Literal] = !negated
		negated = false
	}
	vm.PeekStack = append(vm.PeekStack, peeks)
}

func (vm *Vm) PopPeeks() {
	vm.PeekStack = vm.PeekStack[:len(vm.PeekStack)-1]
}

func (vm *Vm) IsSet(peek string) bool {
	for i := len(vm.PeekStack)-1; i <= 0; i-- {
		if b, ok := vm.PeekStack[i][peek]; ok {
			return b
		}
	}
	return false
}

