package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	pipefish string
	stdlib   string
)

const wikiURL = "https://github.com/tim-hardcastle/pipefish.wiki.git"

const footer = `

## Notes

*This page is automatically generated from the Pipefish standard library. Any edits made directly to this wiki page will be overwritten the next time the documentation is regenerated.*
`

func init() {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	root := filepath.Dir(exe)
	pipefish = filepath.Join(root, "pipefish")
	stdlib = filepath.Join(root, "source", "initializer", "libraries")
}

func main() {
	wiki, cleanup := cloneWiki()
	defer cleanup()
	err := filepath.WalkDir(stdlib, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".pf" {
			return nil
		}
		if err := processFile(path, wiki); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		fatal(err)
	}
	run("git", "-C", wiki, "add", ".")
	diff := exec.Command("git", "-C", wiki, "diff", "--cached", "--quiet")
	err = diff.Run()
	if err == nil {
		fmt.Println("Wiki already up to date.")
		return
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		fatal(err)
	}
	run("git", "-C", wiki, "commit", "-m", "Update generated library documentation")
	run("git", "-C", wiki, "push")
}

func processFile(path, wiki string) error {
	cmd := exec.Command(pipefish, "wiki", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v\n%s", err, output)
	}
	rel, err := filepath.Rel(stdlib, path)
	if err != nil {
		return err
	}
	name := strings.TrimSuffix(rel, ".pf")
	name = filepath.ToSlash(name)
	name = strings.ReplaceAll(name, "/", " ")
	page := filepath.Join(
		wiki,
		"The "+name+" library.md",
	)
	var buf bytes.Buffer
	buf.Write(output)
	if len(output) > 0 && output[len(output)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteString(footer)
	return os.WriteFile(page, buf.Bytes(), 0644)
}

func cloneWiki() (string, func()) {
	tmp, err := os.MkdirTemp("", "pipefish-wiki-*")
	if err != nil {
		fatal(err)
	}
	url := wikiURL
	if token := os.Getenv("WIKI_TOKEN"); token != "" {
		url = fmt.Sprintf(
			"https://x-access-token:%s@github.com/tim-hardcastle/pipefish.wiki.git",
			token,
		)
	}
	run("git", "clone", url, tmp)
	if os.Getenv("WIKI_TOKEN") != "" {
		run("git", "-C", tmp, "config", "user.name", "github-actions[bot]")
		run("git", "-C", tmp, "config", "user.email", "41898282+github-actions[bot]@users.noreply.github.com")
	}
	return tmp, func() {
		_ = os.RemoveAll(tmp)
	}
}

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}