package hub

import (
	"os"
	"regexp"
	"strings"

	"github.com/lmorg/readline/v4"
	"github.com/tim-hardcastle/pipefish/source/text"
	"golang.org/x/term"
)

func (h *Hub) Repl() {
	colonOrEmdash, _ := regexp.Compile(`.*[\w\s]*(:|--)[\s]*$`)
	rline := readline.NewInstance()
	rline.SyntaxHighlighter = func(code []rune) string {
		return h.Services[h.CurrentServiceName()].Highlight(code, h.getFonts())
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
			if left == right {
				handler := func(i int, st *readline.EventState) *readline.EventReturn {
					if len(st.Line) != st.CursorPos {
						if st.Line[st.CursorPos] == right[0] {
							return &readline.EventReturn{
								SetLine:  []rune(st.Line),
								Continue: false,
								SetPos:   st.CursorPos + 1,
							}
						}
					}
					return &readline.EventReturn{
						SetLine:  []rune(st.Line[:st.CursorPos] + right + st.Line[st.CursorPos:]),
						Continue: true,
						SetPos:   st.CursorPos,
					}
				}
				rline.AddEvent(left, handler)
			} else {
				lHandler := func(i int, st *readline.EventState) *readline.EventReturn {
					return &readline.EventReturn{
						SetLine:  []rune(st.Line[:st.CursorPos] + right + st.Line[st.CursorPos:]),
						Continue: true,
						SetPos:   st.CursorPos,
					}
				}
				rline.AddEvent(left, lHandler)
				rHandler := func(i int, st *readline.EventState) *readline.EventReturn {
					if len(st.Line) != st.CursorPos {
						if st.Line[st.CursorPos] == right[0] {
							return &readline.EventReturn{
								SetLine:  []rune(st.Line),
								Continue: false,
								SetPos:   st.CursorPos + 1,
							}
						}
					}
					return &readline.EventReturn{
						SetLine:  []rune(st.Line[:st.CursorPos] + right + st.Line[st.CursorPos:]),
						Continue: true,
						SetPos:   st.CursorPos,
					}
				}
				rline.AddEvent(right, rHandler)
			}
		}
		for {
			rline.SetPrompt(makePrompt(h, ws != ""))
			line, err := rline.ReadlineWithDefault(ws)
			if err == readline.ErrCtrlC {
				print("\nQuit Pipefish? [Y/n] ")
				ch := ReadChar()
				println(string(ch))
				if ch == 'n' || ch == 'N' {
					println(text.Green("OK"))
				} else {
					h.Quit()
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
		sv := h.Services[h.CurrentServiceName()]
		sv.SetOutHandler(sv.MakeTerminalOutHandler())
		h.Do(input, h.TerminalUsername, h.TerminalPassword, h.CurrentServiceName(), false)
	}
}

func ReadChar() rune {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err.Error())
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		panic(err.Error())
	}
	return rune(b[0])
}

func makePrompt(hub *Hub, indented bool) string {
	symbol := PROMPT
	left := hub.CurrentServiceName()
	if indented {
		symbol = INDENT_PROMPT
		left = strings.Repeat(" ", len(left))
	}
	if hub.CurrentServiceName() == "" {
		return symbol
	}
	promptText := text.RESET + text.Cyan(left) + " " + symbol
	if hub.CurrentServiceIsBroken() {
		promptText = text.RESET + text.Red(left) + " " + symbol
	}
	return promptText
}
