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
		{`hub run "../hub/test-files/test.pf"`, `Starting script [36m"test.pf"[39m as service [36m"test"[39m.`},
		{`hub services`, "The hub is running the following services:\n\n[32m  ▪ [0mService [36m\"test\"[39m running script [36m\"test.pf\"[39m."},
		{`foo 2`, `4`},
		{`hub halt "test"`, `[32mOK[0m`},
		{`hub quit`, "[32mOK[0m\n" + hub.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunServiceTest(t, "default", test)
}
