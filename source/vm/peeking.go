package vm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tim-hardcastle/pipefish/source/settings"
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

func (vm *Vm) PushPeeks(peeks map[string]bool) {
	vm.PeekStack = append(vm.PeekStack, peeks)
}

func (vm *Vm) GetPeeksFromTokens(toks []token.Token) map[string]bool {
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
	return peeks
}

func (vm *Vm) PopPeeks() {
	vm.PeekStack = vm.PeekStack[:len(vm.PeekStack)-1]
}

func (vm *Vm) IsSet(peek string) bool {
	for i := len(vm.PeekStack)-1; i >= 0; i-- {
		if b, ok := vm.PeekStack[i][peek]; ok {
			return b
		}
	}
	return false
}

func PeekString(peeks map[string]bool) string {
	result := "'"
	sep := " "
	for k, b := range peeks {
		result = result + sep 
		if !b {
			result = result + "!"
		}
		result = result + k 
		sep = " "
	}
	return result
}

func (vm *Vm) Dump(s string) {
	items := strings.Split(s, "\n")
	result := ""
	for _, item := range items {
		result = result + strings.Repeat("  ", vm.IndentBy) + item + "\n"
	}
	if vm.IsSet("o") {
		file, _ := os.OpenFile(filepath.Join(filepath.FromSlash(settings.PipefishHomeDirectory), settings.DUMP_PATH), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		file.WriteString(result)
		file.Close()
	} else {
		print(result)
	}

}