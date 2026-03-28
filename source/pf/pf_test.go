package pf_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tim-hardcastle/pipefish/source/hub"
	"github.com/tim-hardcastle/pipefish/source/pf"
	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

// We can mostly test the `pf` package by rerunning the tests for the hub, since the hub wraps
// around the `pf` package.

func TestServices(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{"2 + 2", "4"},
		{`hub services`, `The hub isn't running any services.`},
		{`hub run "../hub/test-files/foo.pf"`, `Starting script [36m"foo.pf"[39m as service [36m"foo"[39m.`},
		{`hub services`, "The hub is running the following services:\n\n[32m  ▪ [0mService [36m\"foo\"[39m running script [36m\"foo.pf\"[39m."},
		{`foo 2`, `4`},
		{`hub run "../hub/test-files/bar.pf"`, `Starting script [36m"bar.pf"[39m as service [36m"bar"[39m.`},
		{`bar 2`, `6`},
		{`hub switch "foo"`, `OK`},
		{`foo 2`, `4`},
		{`hub halt "foo"`, `OK`},
		{`hub halt "bar"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + hub.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}
func TestErrors(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{"2 +", "[0] [31mError[39m: can't parse end of line as a prefix at line [33m1:3[39m of REPL input."},
		{`hub why 0`, "\x1b[31mError\x1b[39m: can't parse end of line as a prefix. \n\nYou've put end of line in such a position that it looks like you want it to function as a \x1b[0m\nprefix, but it isn't one. \x1b[0m\n\n                                                      Error has reference \x1b[0m\x1b[48;2;0;0;64m\x1b[97m\"parse/prefix\"\x1b[0m."},
		{`hub where 0`, "2 +\x1b[31m\n\x1b[0m   \x1b[31m▔\x1b[0m"},
		{`hub errors`, "[0] \x1b[31mError\x1b[39m: can't parse end of line as a prefix at line \x1b[33m1:3\x1b[39m of REPL input."},
		
	}
	test_helper.RunHubTest(t, "default", test)
}
func TestEnv(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub env "foo"::42`, `OK`},
		{`hub env delete "foo"`, `OK`},
		{`hub env wipe`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + hub.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}
func TestToGo(t *testing.T) {
	// no t.Parallel()
	wd, _ := os.Getwd()                                  
	pfFile, _ := filepath.Abs(filepath.Join(wd, "/../hub/test-files/togo.pf"))
	srv := pf.NewService()
	srv.InitializeFromFilepath(pfFile)
	pfVal, _ := srv.Do(`42`)
	goVal, _ := srv.ToGo(pfVal, reflect.TypeFor[int]())
	if goVal.(int) != 42 {
		t.Fatal("Can't convert 42.")
	}
	pfVal, _ = srv.Do(`42.0`)
	goVal, _ = srv.ToGo(pfVal, reflect.TypeFor[float64]())
	if goVal.(float64) != 42.0 {
		t.Fatal("Can't convert 42.0.")
	}
	pfVal, _ = srv.Do(`"foo"`)
	goVal, _ = srv.ToGo(pfVal, reflect.TypeFor[string]())
	if goVal.(string) != "foo" {
		t.Fatal("Can't convert `foo`.")
	}
	pfVal, _ = srv.Do(`true`)
	goVal, _ = srv.ToGo(pfVal, reflect.TypeFor[bool]())
	if goVal.(bool) != true {
		t.Fatal("Can't convert true.")
	}
}
