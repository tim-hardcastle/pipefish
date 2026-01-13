package hub

import (
	"regexp"
	"strings"

	"github.com/lmorg/readline/v4"
	"github.com/tim-hardcastle/pipefish/source/text"
)

// TODO --- once the highlighting is semantic and not syntactic, we'll
// need a different highlighter for each service.
func (hub *Hub) Repl() {
	colonOrEmdash, _ := regexp.Compile(`.*[\w\s]*(:|--)[\s]*$`)
	rline := readline.NewInstance()
	rline.SyntaxHighlighter = func(code []rune) string {
		return hub.Services[hub.currentServiceName()].Highlight(code, hub.getFonts())
	}
	for {

		ws := ""
		input := ""
		c := 0
		PAIRS := [][2]string{
			{"(", ")"},
			{"{", "}"},
			{"[", "]"},
			{"\"", "\""},
			{"`", "`"},
			{"|", "|"},
		}
		for _, pair := range PAIRS {
			left := pair[0]
			right := pair[1]
			handler := func(i int, st *readline.EventState) *readline.EventReturn {
				return &readline.EventReturn{
					SetLine:  []rune(st.Line[:st.CursorPos] + right + st.Line[st.CursorPos:]),
					Continue: true,
					SetPos:   st.CursorPos,
				}
			}
			rline.AddEvent(left, handler)
		}
		for {
			rline.SetPrompt(makePrompt(hub, ws != ""))
			line, err := rline.ReadlineWithDefault(ws)
			if err == readline.ErrCtrlC {
				print("\nQuit Pipefish? [Y/n] ")
				ch := text.ReadChar()
				println(string(ch))
				if ch == 'n' || ch == 'N' {
					println(text.Green("OK"))
				} else {
					hub.Quit()
					return
				}
			}
			c++
			input = input + line + "\n"
			ws = ""
			for _, c := range line {
				if c == ' ' || c == '\t' {
					ws = ws + string(c)
				} else {
					break
				}
			}
			if colonOrEmdash.Match([]byte(line)) {
				ws = ws + "  "
			}
			if ws == "" {
				break
			}
		}
		input = strings.TrimSpace(input)
		sv := hub.Services[hub.currentServiceName()]
		sv.SetOutHandler(sv.MakeTerminalOutHandler())
		_, quit := hub.Do(input, hub.TerminalUsername, hub.TerminalPassword, hub.currentServiceName(), false)
		if quit {
			break
		}
	}
}

func makePrompt(hub *Hub, indented bool) string {
	symbol := PROMPT
	left := hub.currentServiceName()
	if indented {
		symbol = INDENT_PROMPT
		left = strings.Repeat(" ", len(left))
	}
	if hub.currentServiceName() == "" {
		return symbol
	}
	promptText := text.RESET + text.Cyan(left) + " " + symbol
	if hub.CurrentServiceIsBroken() {
		promptText = text.RESET + text.Red(left) + " " + symbol
	}
	return promptText
}
