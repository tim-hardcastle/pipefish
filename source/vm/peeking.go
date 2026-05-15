package vm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/token"
)

// This contains temporary or permanent code for peeking at the operations of the VM.

// We read in the operations.md file and use it as a source of truth for dumping the compiler and VM, and
// for sanity checks.
type operatorInfo struct {
	opcode string
	operandFlavors []string
	description string 
	notes []string // We keep these as seperate lines so we can add them as comments to vm.go.
}

var opInfo = map[Opcode]operatorInfo{}

func init() {
	content, _ := os.ReadFile(filepath.Join(settings.PipefishHomeDirectory, "source/vm/operations.md"))
	lines := strings.Split(string(content), "\n")
	i := 0
	for ; lines[i] != "## Operators"; i ++ {} // Skips the preamble to `operations.md`.
	i++
	for ; i < len(lines) ; {
		// We start off at a newline, which we skip.
		i++
		headline := lines[i]
		fields := strings.Fields(headline)
		opcode := fields[0]
		operands := fields[2:]
		i++
		description := lines[i]
		i++
		notes := []string{}
		for ; i < len(lines) && lines[i] != ""; i++ {
			notes = append(notes, lines[i])
		} 
		opInfo[OPCODES[opcode]] = operatorInfo{
			opcode: opcode,
			operandFlavors: operands,
			description: description,
			notes: notes,
		}
	}
}

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

var OPCODES = map[string]Opcode{
        "addf": Addf, 
        "addi": Addi, 
        "addL": AddL, 
        "addS": AddS, 
        "adds": Adds, 
        "adrs": Adrs, 
        "adsr": Adsr, 
        "adtk": Adtk, 
        "andb": Andb, 
        "aref": Aref, 
        "asgm": Asgm, 
        "auto": Auto, 
        "call": Call,  
        "calt": CalT,  
        "casP": CasP, 
        "cast": Cast, 
        "casx": Casx, 
        "cc11": Cc11, 
        "cc1T": Cc1T, 
        "ccT1": CcT1, 
        "ccTT": CcTT, 
        "ccxx": Ccxx, 
        "chck": Chck, 
        "chrf": Chrf, 
        "clon": Clon, 
        "cpnt": Cpnt, 
        "conL": ConL, 
        "conS": ConS, 
        "cv1T": Cv1T, 
        "cvTT": CvTT, 
        "diif": Diif, 
        "divf": Divf, 
        "divi": Divi, 
        "dvfi": Dvfi, 
        "dvif": Dvif, 
        "dofn": Dofn, 
        "dref": Dref, 
        "equb": Equb, 
        "equf": Equf, 
        "equi": Equi, 
        "equs": Equs, 
        "equt": Equt, 
        "eqxx": Eqxx, 
        "eval": Eval, 
        "extn": Extn,  
        "flpp": Flpp, 
        "flps": Flps, 
        "flti": Flti, 
        "flts": Flts, 
        "gofn": Gofn,  
        "gsql": Gsql, 
        "gtef": Gtef, 
        "gtei": Gtei, 
        "gthf": Gthf, 
        "gthi": Gthi, 
        "idxL": IdxL, 
        "idxp": Idxp, 
        "idxs": Idxs, 
        "idxT": IdxT, 
        "ixTn": IxTn, 
        "ixZl": IxZl, 
        "ixZn": IxZn, 
        "inpt": Inpt, 
        "inxL": InxL, 
        "inxS": InxS, 
        "inxt": Inxt, 
        "inxT": InxT, 
        "inte": Inte, 
        "intf": Intf, 
        "ints": Ints, 
        "itgk": Itgk, 
        "itgv": Itgv, 
        "itkv": Itkv, 
        "itor": Itor, 
        "ixSn": IxSn, 
        "ixXx": IxXx, 
        "jmp" : Jmp, 
        "json": Json, 
        "jsr" : Jsr, 
        "keyM": KeyM, 
        "keyZ": KeyZ, 
        "lbls": Lbls, 
        "lenL": LenL, 
        "lenM": LenM, 
        "lens": Lens, 
        "lenS": LenS, 
        "lenT": LenT, 
        "litx": Litx, 
        "list": List, 
        "lnSn": LnSn, 
        "logn": Logn, 
        "logy": Logy, 
        "mkEn": MkEn, 
        "mker": Mker, 
        "mkfn": Mkfn, 
        "mkit": Mkit,  
        "mkmp": Mkmp, 
        "mkpr": Mkpr, 
        "mkSn": MkSn, 
        "mkst": Mkst, 
        "modi": Modi, 
        "mpar": Mpar, 
        "mulf": Mulf, 
        "muli": Muli, 
        "negf": Negf, 
        "negi": Negi, 
        "notb": Notb, 
        "outp": Outp, 
        "outt": Outt, 
        "qabt": Qabt, 
        "qfls": Qfls, 
        "qitr": Qitr, 
        "qleT": QleT, 
        "qlnT": QlnT, 
        "qlog": Qlog, 
        "qnab": Qnab, 
        "qntp": Qntp, 
        "qsat": Qsat, 
        "qsnq": Qsnq, 
        "qtpt": Qtpt, 
        "qtru": Qtru, 
        "qtyp": Qtyp, 
        "psql": Psql, 
        "ret" : Ret, 
        "rpop": Rpop, 
        "rpsh": Rpsh, 
        "sliL": SliL, 
        "slis": Slis, 
        "sliT": SliT, 
        "slTn": SlTn, 
        "strc": Strc, 
        "strP": StrP, 
        "strx": Strx, 
        "subf": Subf, 
        "subi": Subi, 
        "subS": SubS, 
        "thnk": Thnk, 
        "tinf": Tinf, 
        "tupf": Tplf, 
        "trak": Trak, 
        "tupL": TupL, 
        "tuLx": TuLx, 
        "typu": Typu, 
        "typx": Typx, 
        "untE": UntE, 
        "untk": Untk, 
        "uwrp": Uwrp, 
        "vlid": Vlid, 
        "wrHb": WrHb, 
        "wthL": WthL, 
        "wthM": WthM, 
        "wthT": WthT, 
        "wthZ": WthZ, 
        "wtoM": WtoM, 
        "yeet": Yeet, 
}