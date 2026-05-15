package vm

import "strconv"

func MakeOp(oc Opcode, args ...uint32) *Operation {
	return &Operation{Opcode: oc, Args: args}
}

type Operation struct {
	Opcode Opcode
	Args   []uint32
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

func (op *Operation) MakeLastArg(loc uint32) {
	op.Args[len(op.Args)-1] = loc
}

type Opcode uint8

const (
	Addf Opcode = iota
	Addi
	AddL
	AddS
	Adds
	Adrs
	Adsr
	Adtk
	Andb
	Aref
	Asgm
	Auto
	Call
	CalT
	CasP
	Cast
	Casx
	Cc11
	Cc1T
	CcT1
	CcTT
	Ccxx
	Chck
	Chrf
	Clon
	ConL
	ConS
	CoSn
	Cpnt
	Cv1T
	CvTT
	Diif
	Divf
	Divi
	Dofn
	Dref
	Dvfi
	Dvif
	Equb
	Equf
	Equi
	Equs
	Equt
	Eqxx
	Eval
	Extn
	Flpp
	Flps
	Flti
	Flts
	Gofn
	Gsql
	Gtef
	Gtei
	Gthf
	Gthi
	IctS
	IdxL
	IdxM
	Idxp
	Idxs
	IdxT
	Inpt
	InxL
	InxS
	Inxt
	InxT
	Inte
	Intf
	Ints
	Itgk
	Itkv
	Itgv
	Itor
	IxSn
	IxTn
	IxXx
	IxZl
	IxZn
	Jmp
	Json
	Jsr
	KeyM
	KeyZ
	Lbls
	LenL
	Lens
	LenM
	LenS
	LenT
	List
	Litx
	LnSn
	Logn
	Logy
	Mker
	Mkfn
	Mkit
	MkEn
	Mkmp
	Mkpr
	MkSn
	Mkst
	Mlfi
	Modi
	Mulf
	Muli
	Negf
	Negi
	Notb
	Outp
	Outt
	Psql
	Mpar
	Qabt
	Qfls
	Qitr
	QleT
	QlnT
	Qlog
	Qnab
	Qntp
	Qsat
	Qsnq
	Qspt
	Qspq
	Qtpt
	Qtru
	Qtyp
	Ret
	Rpop
	Rpsh
	SliL
	Slis
	SliT
	SlTn
	Strc
	StrP
	Strx
	Subf
	Subi
	SubS
	Thnk
	Tinf
	Tplf
	Trak
	TupL
	TuLx
	Typu
	Typx
	UntE
	Untk
	Uwrp
	Vlid
	WrHb
	WthL
	WthM
	WthT
	WthZ
	WtoM
	Yeet
)
