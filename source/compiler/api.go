package compiler

import (
	"strings"

	"github.com/tim-hardcastle/pipefish/source/text"
	"github.com/tim-hardcastle/pipefish/source/values"
)

// This supplies the bits and pieces we need to render the API.
// We're doing this here and now rather than at initialization so that in principle
// we could get the font and width from a desktop client.

func (cp *Compiler) Api(name string, path []string, fonts values.Map, width int) string {
	markdowner := text.NewMarkdown("", width, func(s string) string { return cp.Highlight([]rune(s), fonts) })
	return cp.RenderApi(name, path, fonts, markdowner)
}

func (cp *Compiler) Wiki(path []string) string {
	return cp.RenderApi("", path, values.Map{}, wikifier{})
}

func (cp *Compiler) RenderApi(name string, path []string, fonts values.Map, rdr renderer) string {
	_, md := rdr.(*text.Markdown)
	if len(path) > 0 {
		newCp, ok := cp.Modules[path[0]]
		if !ok {
			return rdr.Render([]string{"The module `" + path[0] + "` does not exist."})
		}
		if newCp.P.Private {
			return rdr.Render([]string{"The module `" + path[0] + "` is private."})
		}
		return newCp.RenderApi(name, path[1:], fonts, rdr)
	}
	hasContents := false
	result := ""
	if name != "" {
		result = rdr.Render([]string{"# " + name})
		result = result + "\n"
	}
	if cp.DocString != "" {
		result = result + rdr.Render([]string{"## Overview"})
		result = result + "\n"
		result = result + rdr.Render(strings.Split(cp.DocString, "\n"))
		result = result + "\n"
	}
	for i, items := range cp.ApiDescription {
		if len(items) == 0 {
			continue
		}
		hasContents = true
		result = result + rdr.Render([]string{"## " + headings[i]})
		for _, item := range items {
			heading := item.Declaration
			if item.DocString != "" && md {
				heading = append(heading, ' ', ':')
			}
			if md {
				result = result + "\n" + text.Cyan("•") + " " + cp.Highlight(heading, fonts) + "\n"
			} else {
				result = result + "\n### `" + string(heading) + "`\n"
			}
			if item.DocString != "" {
				result = result + "\n" + rdr.Render(strings.Split(item.DocString,"\n"))
			}
		}
	}
	if !hasContents {
		result = result + "Nothing has been declared.\n"
	}
	return result + "\n"
}

type ApiItem struct {
	Declaration []rune
	DocString   string
}

var headings = []string{"Modules", "Types", "Constants", "Variables", "Commands", "Functions"}

type renderer interface {
	Render([]string) string
}

type wikifier struct{}

func (w wikifier) Render(lines []string) string {
	result :=  ""
	lastWasList := false
	for i, line := range lines {
		if text.Head(line, "- ") {
			if !lastWasList {
				result = result + "\n"
				lastWasList = true 
			}
			result = result + line + "\n"
		} else {
			if line == "" && i != len(lines)-1 {
				result = result + "\n\n"
			} else {
				result = result + line + " "
			}
			lastWasList = false
		}
	}
	return result + "\n"
}

