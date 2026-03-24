package text_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/text"
	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

func TestColors(t *testing.T) {
	if ! (text.Red("foo") == "[31mfoo[0m") {
		t.Fatalf("Can't make things red.", )
	}
	if ! (text.Cyan("foo") == "[36mfoo[0m") {
		t.Fatalf("Can't make things cyan.")
	}
	if ! (text.Green("foo") == "[32mfoo[0m") {
		t.Fatalf("Can't make things green.")
	}
	if ! (text.Yellow("foo") == "[33mfoo[0m") {
		t.Fatalf("Can't make things green.")
	}
	if ! (text.Emph("foo") == "`foo`") {
		t.Fatalf("Can't emphasize things.")
	}
	if ! (text.ErrorFont("foo") == "[38;2;244;71;71m[4mfoo[0m") {
		t.Fatalf("Can't make error font.")
	}
}

func TestMarkdown(t *testing.T) {
	tests := []test_helper.TestItem{
		{`Hello`, `Hello `},
		{`Hello *darkness* my **old** ***friend***.`, `Hello [3mdarkness[22m[23m my [1mold[22m[23m [1m[3mfriend[22m[23m. `},
		{`<R>red</> <B><blue</>`, `[31mred[39m [34m<blue[39m`},
		{"inline `code` looks like this", `inline [0m[48;2;0;0;64m[97mcode[0m looks like this `},
		{`## Heading`, "[1m════ Heading ══════════════════════════════════════════════════════════════════════════════\n[0m"},
		{"> Block quote", "\n[0m  ‖ Block quote "},
		{"- Bullet point", "\n[0m  ▪ Bullet point "},
		{"```\ncode\nmore code\n```", "\n  ¦ code\n  ¦ more code\n"},
	}
	md := text.NewMarkdown("", 92, func(s string) string {return s})
	for _, test := range tests {
		got := md.RenderString(test.Input)
		println(got)
		if !(test.Want == got) {
			t.Fatalf("Test failed with input %s \nExp :\n%s\nGot :\n%s", test.Input, test.Want, got)
		}
	}
}

func TestTextUtils(t *testing.T) {
	if ! (text.Flatten("foo/bar.troz") == "foo_bar_troz") {
		t.Fatalf("Flatten failed")
	}
	if ! (text.Head("foolish", "foo")) {
		t.Fatalf("Head failed")
	}
	if (text.Head("aardvark", "foo")) {
		t.Fatalf("Head failed")
	}
	if (text.Head("aa", "foo")) {
		t.Fatalf("Head failed")
	}
	if ! (text.Tail("proof", "oof")) {
		t.Fatalf("Tail failed")
	}
	if (text.Tail("aardvark", "oof")) {
		t.Fatalf("Tail failed")
	}
	if (text.Tail("aa", "oof")) {
		t.Fatalf("Tail failed")
	}
}
