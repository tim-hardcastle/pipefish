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
	if len(path) > 0 {
		newCp, ok := cp.Modules[path[0]]
		if !ok {
			return markdowner.Render([]string{"The module `" + path[0] + "` does not exist."})
		}
		if newCp.P.Private {
			return markdowner.Render([]string{"The module `" + path[0] + "` is private."})
		}
		return newCp.Api(name, path[1:], fonts, width)
	}
	hasContents := false
	result := ""
	if name != "" || cp.DocString != "" {
		title := "# " + name
		result = "\n" + markdowner.Render([]string{title})
	}
	if cp.DocString != "" {
		result = result + "\n"
		result = result + markdowner.Render([]string{cp.DocString})
		result = result + "\n"
	}
	for i, items := range cp.ApiDescription {
		if len(items) == 0 {
			continue
		}
		hasContents = true
		result = result + "\n" + markdowner.Render([]string{"### " + headings[i]})
		for _, item := range items {
			heading := item.Declaration
			if item.DocString != "" {
				heading = append(heading, ' ', ':')
			}
			result = result + "\n" + text.Cyan("•") + " " + cp.Highlight(heading, fonts) + "\n"
			if item.DocString != "" {
				result = result + "\n" + markdowner.Render(strings.Split(item.DocString,"\n"))
			}
		}
	}
	if !hasContents {
		result = result + "\nNothing has been declared.\n\n"
		return result
	}
	return result + "\n"
}

type ApiItem struct {
	Declaration []rune
	DocString   string
}

var headings = []string{"Modules", "Types", "Constants", "Variables", "Commands", "Functions"}
