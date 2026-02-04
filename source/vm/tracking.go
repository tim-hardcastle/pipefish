package vm

import (
	"bytes"
	"strconv"
	"time"

	"github.com/tim-hardcastle/pipefish/source/text"
	"github.com/tim-hardcastle/pipefish/source/token"
	"github.com/tim-hardcastle/pipefish/source/values"
)

// This file supplies resources for generating tracking information at runtims.

type TrackingData struct {
	Flavor     TrackingFlavor
	Tok        *token.Token
	LogToLoc   uint32 // The memory location in the compiler storing what to do with the logging output.
	LogTimeLoc uint32 // The memory location in the compiler storing whether to attach the time to the logging output.
	Args       []any
}

type TrackingFlavor int

const (
	TR_CONDITION TrackingFlavor = iota
	TR_ELSE
	TR_FNCALL
	TR_LITERAL
	TR_RESULT
	TR_RETURN
)

func (vm *Vm) trackingIs(i int, tf TrackingFlavor) bool {
	if i < 0 || i >= len(vm.LiveTracking) {
		return false
	}
	return vm.LiveTracking[i].Flavor == tf
}

func (vm *Vm) TrackingToString(tdL []TrackingData) string {
	time := time.Now()
	if len(tdL) == 0 {
		return ("\nNo tracking data exists.\n")
	}
	var out bytes.Buffer
	for i, td := range tdL {
		logTime := vm.Mem[td.LogTimeLoc].V.(bool)
		args := td.Args
		switch td.Flavor {
		case TR_CONDITION:
			out.WriteString("At ")
			if logTime {
				out.WriteString(time.Format("15:04:05"))
				out.WriteString(", at ")
			}
			out.WriteString("line ")
			out.WriteString(strconv.Itoa(td.Tok.Line))
			out.WriteString(" we evaluated the condition ")
			out.WriteString(text.Emph(args[0].(string)))
			out.WriteString(". ")
		case TR_ELSE:
			out.WriteString("At ")
			if logTime {
				out.WriteString(time.Format("15:04:05"))
				out.WriteString(", at ")
			}
			out.WriteString("line ")
			out.WriteString(strconv.Itoa(td.Tok.Line))
			out.WriteString(" we took the ")
			out.WriteString(text.Emph("else"))
			out.WriteString(" branch")
			if !vm.trackingIs(i+1, TR_RETURN) {
				out.WriteString(".\n")
			}
		case TR_FNCALL:
			if logTime {
				out.WriteString("At ")
				out.WriteString(time.Format("15:04:05"))
				out.WriteString(", w")
			} else {
				out.WriteString("W")
			}
			out.WriteString("e called function ")
			out.WriteString(text.Emph(args[0].(string)))
			out.WriteString(" - defined at line ")
			out.WriteString(strconv.Itoa(td.Tok.Line))
			out.WriteString(" ")
			if len(args) > 1 {
				out.WriteString("- with ")
				sep := ""
				for i := 1; i < len(args); i = i + 2 {
					out.WriteString(sep)
					out.WriteString(text.Emph(args[i].(string)) + " = " + text.Emph(vm.Literal(args[i+1].(values.Value), 0)))
					sep = ", "
				}
			}
			out.WriteString(".\n")
		case TR_LITERAL:
			if logTime {
				out.WriteString("At ")
				out.WriteString(time.Format("15:04:05"))
				out.WriteString(", l")
			} else {
				out.WriteString("L")
			}
			out.WriteString("og at line ")
			out.WriteString(strconv.Itoa(td.Tok.Line))
			out.WriteString(" : ")
			out.WriteString(args[0].(values.Value).V.(string))
			out.WriteString("\n")
		case TR_RESULT:
			if logTime {
				out.WriteString("At ")
				out.WriteString(time.Format("15:04:05"))
				out.WriteString(", t")
			} else {
				out.WriteString("T")
			}
			if args[0].(values.Value).V.(bool) {
				out.WriteString("he condition succeeded.\n")
			} else {
				out.WriteString("he condition failed.\n")
			}
		case TR_RETURN:
			if args[1].(values.Value).T != values.UNSATISFIED_CONDITIONAL {
				if vm.trackingIs(i-1, TR_ELSE) {
					out.WriteString(", so at ")
				} else {
					out.WriteString("At ")
				}
				if logTime {
					out.WriteString(time.Format("15:04:05"))
					out.WriteString(", at ")
				}
				out.WriteString("line ")
				out.WriteString(strconv.Itoa(td.Tok.Line))
				out.WriteString(" ")
				//out.WriteString("of ")
				//out.WriteString(text.Emph(td.tok.Source))
				//out.WriteString(" ")
				out.WriteString("function ")
				out.WriteString(text.Emph(args[0].(string)))
				out.WriteString(" returned ")
				out.WriteString(vm.Literal(args[1].(values.Value), 0))
				out.WriteString(".\n")
			}
		}
	}
	return out.String()
}
