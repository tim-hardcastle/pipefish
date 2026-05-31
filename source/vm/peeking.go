package vm

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/token"
)

// This contains temporary or permanent code for peeking at the operations of the VM.

// We read in the operations.md file and use it as a source of truth for dumping the compiler and VM, and
// for sanity checks.
type operatorInfo struct {
	opcode         string
	operandFlavors []string
	description    string
	notes          []string // We keep these as seperate lines so we can add them as comments to vm.go.
}

var opInfo = map[Opcode]operatorInfo{}

func init() {
	// Set up the operations map.
	content, _ := os.ReadFile(filepath.Join(settings.PipefishHomeDirectory, "source/vm/operations.md"))
	lines := strings.Split(string(content), "\n")
	i := 0
	for ; lines[i] != "## Operators"; i++ {
	} // Skips the preamble to `operations.md`.
	i++
	for i < len(lines) {
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
			opcode:         opcode,
			operandFlavors: operands,
			description:    description,
			notes:          notes,
		}
	}
	// Add comments to operations.go.
	if !testing.Testing() {
		operationsFile := filepath.Join(settings.PipefishHomeDirectory, "source/vm/operations.go")
		content, _ = os.ReadFile(operationsFile)
		lines = strings.Split(string(content), "\n")
		result := ""
		i = 0
		for ; lines[i] != "const ("; i++ {
			result = result + lines[i] + "\n"
		}
		result = result + "const (\n"
		i++
		commentRegexp, _ := regexp.Compile(`^\s*//*.`)
		for ; lines[i] != ")"; i++ {
			if commentRegexp.MatchString(lines[i]) || lines[i] == "\t" {
				continue
			}
			opcodeEnd := strings.Index(lines[i], " ")
			if opcodeEnd == -1 {
				opcodeEnd = len(lines[i])
			}
			opcode := lines[i][1:opcodeEnd]
			runes := []rune(opcode)
			runes[0] = unicode.ToLower(runes[0])
			opcode = string(runes)
			opNumber := OPCODES[opcode]
			result = result + "\t// " + opInfo[opNumber].description + " (" + strings.Join(opInfo[opNumber].operandFlavors, " ") + ")\n"
			result = result + lines[i] + "\n"
		}
		result = result + ")\n"
		os.WriteFile(operationsFile, []byte(result), 0666)

		// Now the comments to vm.go.
		caseRegexp, _ := regexp.Compile(`(\t\t\tcase [A-Z][A-Za-z1]{2,3}:)\s*(|//.*)`)
		vmFile := filepath.Join(settings.PipefishHomeDirectory, "source/vm/vm.go")
		content, _ = os.ReadFile(vmFile)
		lines = strings.Split(string(content), "\n")
		result = ""
		eatComments := false
		for i = 0; i < len(lines); i++ {
			line := lines[i]
			if eatComments && commentRegexp.MatchString(line) {
				continue
			}
			eatComments = false
			if match := caseRegexp.FindString(line); match != "" {
				line = caseRegexp.ReplaceAllString(match, `$1`)
				opcode := line[8 : len(line)-1]
				runes := []rune(opcode)
				runes[0] = unicode.ToLower(runes[0])
				opcode = string(runes)
				opNumber := OPCODES[opcode]
				result = result + line + " // " + opInfo[opNumber].description + " (" + strings.Join(opInfo[opNumber].operandFlavors, " ") + ")\n"
				for _, noteLine := range opInfo[opNumber].notes {
					result = result + "\t\t\t\t// " + noteLine + "\n"
				}
				eatComments = true
				continue
			}
			result = result + line + "\n"
		}
		os.WriteFile(vmFile, []byte(result), 0666)
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
	for i := len(vm.PeekStack) - 1; i >= 0; i-- {
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

func (op *Operation) ppOperand(i int) string {
	// If we're calling this, the OPERANDS table shows that the operation ought to have an [i] operand.
	//
	opFlavor := opInfo[op.Opcode].operandFlavors[i]
	if i >= len(op.Args) {
		if opFlavor == "tup" {
			return " ()"
		}
		println("Not enough operands supplied to " + opInfo[op.Opcode].opcode + "; was expecting " + strconv.Itoa(len(opInfo[op.Opcode].operandFlavors)) + " but got " + strconv.Itoa(len(op.Args)) + ".")
		argStr := "Args were:"
		for j := range op.Args {
			argStr = argStr + " " + op.ppOperand(j)
		}
		argStr = argStr + "."
		println(argStr)
		panic("That's all folks!")
	}
	opVal := strconv.Itoa(int(op.Args[i]))
	switch opFlavor {
	case "chk":
		return "?" + opVal
	case "dst":
		return " m" + opVal + " <-"
	case "gfn":
		return " Γ" + opVal
	case "lfc":
		return " Λ" + opVal
	case "loc":
		return " @" + opVal
	case "mem":
		return " m" + opVal
	case "num":
		return " %" + opVal
	case "ptp":
		return " {" + opVal + "}"
	case "sfc":
		return " Σ" + opVal
	case "trk":
		return " ~" + opVal
	case "tok":
		return " TK" + opVal
	case "tup":
		args := op.Args[i : len(op.Args)+1-len(opInfo[op.Opcode].operandFlavors)+i]
		result := " ("
		for j, v := range args[:] {
			result = result + "m" + strconv.Itoa(int(v))
			if j < len(args)-1 {
				result = result + " "
			}
		}
		return result + ")"
	case "typ":
		return " t" + opVal
	}
	panic("Unknown operand type '" + opFlavor + "'")
}

func describe(op *Operation) string {
	operands := opInfo[op.Opcode].operandFlavors
	result := opInfo[op.Opcode].opcode
	for i := range operands {
		result = result + op.ppOperand(i)
	}
	return result + "  // " + opInfo[op.Opcode].description + "."
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
	"jmp":  Jmp,
	"json": Json,
	"jsr":  Jsr,
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
	"ret":  Ret,
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
