package hub_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tim-hardcastle/pipefish/source/hub"
)

func TestHub(t *testing.T) {
	test := []testPair{
		{"2 + 2", "4"},
	}
	runTest(t, "default", test)
}

type testPair struct {
	input  string
	expect string
}

type capturingWriter struct {capture string} 

func (c *capturingWriter) get() string {
	s := c.capture 
	c.capture = ""
	return s
}

func (c *capturingWriter) Write(b []byte) (n int, err error) {
	c.capture = c.capture + string(b)
	return len(b), nil
}

func runTest(t *testing.T, hubName string, test []testPair) { 
	wd, _ := os.Getwd() // The working directory is the directory containing the package being tested.
	sourceDir, _ := filepath.Abs(filepath.Join(wd, "/../")) // We may be calling this either from in the `hub` direcotry or `pf`.
	hubDir := filepath.Join(sourceDir, "hub/test-files", hubName)
	h := hub.New(hubDir, &capturingWriter{})
	for _, item := range test {
		h.Do(item.input, "", "", "", false)
		result := h.Out.(*capturingWriter).get()
		if result != item.expect + "\n" {
			t.Fatal("\nOn input" + item.input + "\n    Exp : " + item.expect + "\n    Got :" + result)
		}
	}
}