//
// Pipefish version 0.7.0
//
// Acknowledgments
//
// I began with Thorsten Ball’s Writing An Interpreter In Go (https://interpreterbook.com/) and the
// accompanying code, and although his language and mine differ very much in their syntax, semantics,
// implementation, and ambitions, I still owe him a considerable debt.
//
// I owe thanks to Laurence Morgan (lmorg on Github) for adding features to his readline library for
// my sake.
//
// Much gratitude is due to r/programminglanguages collectively for advice and encouragement.
//

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tim-hardcastle/pipefish/source/hub"
	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/text"
)

func main() {
	if len(os.Args) == 1 {
		showhelp()
		return
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			showhelp()
			return
		case "-v", "--version", "version":
			os.Stdout.WriteString("\nPipefish version " + text.VERSION + ".\n\n")
			return
		case "-r", "--run", "run":
			hub.StartServiceFromCli()
		case "-t", "--tui", "tui": // Left blank to avoid the default.
		case "-w", "--w", "wiki":  
			hub.GetWiki()
		default:
			os.Stdout.WriteString("\nPipefish doesn't recognize the command '" + os.Args[1] + "'.\n")
			println()
			showhelp()
			os.Exit(1)
		}
	}

	fmt.Print(text.Logo())
	bytes, _ := os.ReadFile(filepath.Join(settings.PipefishHomeDirectory, ("user/hub.dat")))
	filename := string(bytes)
	if filepath.IsLocal(filename) {
		filepath.Join(settings.PipefishHomeDirectory, filename)
	}
	h := hub.New(filename, os.Stdout)
	h.Repl()
}

func showhelp() {
	os.Stdout.WriteString(hub.HELP)
}
