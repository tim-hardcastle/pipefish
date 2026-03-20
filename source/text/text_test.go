package text_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/text"
	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

func TestMarkdown(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Hello`, `Hello `},
		{`Hello *darkness* my **old** ***friend***.`, `Hello [3mdarkness[22m[23m my [1mold[22m[23m [1m[3mfriend[22m[23m. `},
		{`<R>red</> <B><blue</>`, `[31mred[39m [34m<blue[39m`},
		{"inline `code` looks like this", `inline [0m[48;2;0;0;64m[97mcode[0m looks like this `},
		{`## Heading`, "[1m════ Heading ══════════════════════════════════════════════════════════════════════════════\n[0m"},
		{`> Block quote`, "\n[0m  ‖ Block quote "},
		{`- Bullet point`, "\n[0m  ▪ Bullet point "},
	}
	md := text.NewMarkdown("", 92, func(s string) string {return s})
	for _, test := range tests {
		got := md.Render([]string{test.Input})
		if !(test.Want == got) {
			t.Fatalf("Test failed with input %s \nExp :\n%s\nGot :\n%s", test.Input, test.Want, got)
		}
	}
}