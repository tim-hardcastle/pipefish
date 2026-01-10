package pf_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/hub"
	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

// We can mostly test the `pf` package by rerunning the tests for the hub, since the hub wraps
// around the `pf` package.

func TestServices(t *testing.T) {
	test := []test_helper.TestItem{
		{"2 + 2", "4"},
		{`hub services`, `The hub isn't running any services.`},
		{`hub run "../hub/test-files/foo.pf"`, `Starting script [36m"foo.pf"[39m as service [36m"foo"[39m.`},
		{`hub services`, "The hub is running the following services:\n\n[32m  ▪ [0mService [36m\"foo\"[39m running script [36m\"foo.pf\"[39m."},
		{`foo 2`, `4`},
		{`hub run "../hub/test-files/bar.pf"`, `Starting script [36m"bar.pf"[39m as service [36m"bar"[39m.`},
		{`bar 2`, `6`},
		{`hub switch "foo"`, `[32mOK[0m`},
		{`foo 2`, `4`},
		{`hub halt "foo"`, `[32mOK[0m`},
		{`hub halt "bar"`, `[32mOK[0m`},
		{`hub quit`, "[32mOK[0m\n" + hub.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunServiceTest(t, "default", test)
}
func TestEnv(t *testing.T) {
	test := []test_helper.TestItem{
		{`hub env "foo"::42`, `[32mOK[0m`},
		{`hub env delete "foo"`, `[32mOK[0m`},
		{`hub env wipe`, `[32mOK[0m`},
		{`hub quit`, "[32mOK[0m\n" + hub.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunServiceTest(t, "default", test)
}