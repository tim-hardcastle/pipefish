package text

// This consists of a bunch of text utilities to help in generating pretty and meaningful
// help messages, error messages, etc.

// As a result of factoring out the pf library, it has some overlap with functions declared
// in the `hub` package, and changes made here may need to be reflected there.

import (
	"strings"
)

func Flatten(s string) string {
	s = strings.Replace(s, ".", "_", -1)
	s = strings.Replace(s, "/", "_", -1)
	return s
}

func Cyan(s string) string {
	return CYAN + s + RESET
}

func Emph(s string) string {
	return "`" + s + "`"
}

func Red(s string) string {
	return RED + s + RESET
}

func Green(s string) string {
	return GREEN + s + RESET
}

func Yellow(s string) string {
	return YELLOW + s + RESET
}

func ErrorFont(s string) string {
	return BAD_RED + UNDERLINE + s + RESET
}

const (
	RESET                  = "\033[0m"
	RESET_FOREGROUND       = "\033[39m"
	RESET_BACKGROUND       = "\033[49m"
	RESET_BOLD             = "\033[22m"
	RESET_ITALIC           = "\033[23m"
	RESET_UNDERLINE        = "\033[24m"
	UNDERLINE              = "\033[4m"
	RED                    = "\033[31m"
	BAD_RED                = "\033[38;2;244;71;71m"
	YELLOW                 = "\033[33m"
	GREEN                  = "\033[32m"
	BLUE                   = "\033[34m"
	PURPLE                 = "\033[35m"
	CYAN                   = "\033[36m"
	GRAY                   = "\033[37m"
	INLINE_CODE_BACKGROUND = "\033[48;2;0;0;64m"
	WHITE                  = "\033[97m"
	ITALIC                 = "\033[3m"
	BOLD                   = "\033[1m"
	BULLET                 = "  ▪ "
	MASK                   = '▪'
	RT_ERROR               = "<R>Error</>: "
	ERROR                  = "<R>Error</>: "
	ORANGE                 = "\033[38;2;255;165;0m"
)

func Head(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	return s[:len(substr)] == substr
}

func Tail(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	return s[len(s)-len(substr):] == substr
}

