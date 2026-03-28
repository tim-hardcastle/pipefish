package hub_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/hub"
	"github.com/tim-hardcastle/pipefish/source/test_helper"
)

func TestServices(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{"2 + 2", "4"},
		{`hub services`, `The hub isn't running any services.`},
		{`hub run "../hub/test-files/foo.pf"`, `Starting script [36m"foo.pf"[39m as service [36m"foo"[39m.`},
		{`hub reset`, "Restarting script \x1b[36m\"../hub/test-files/foo.pf\"\x1b[39m as service \x1b[36m\"foo\"\x1b[39m."},
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

func TestBrokenService(t *testing.T) { // We want to make sure that if the service is broken, queries get handed off to the empty service.
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/broken.pf"`, "Starting script [36m\"broken.pf\"[39m as service [36m\"broken\"[39m. [0] [31mError[39m: unexpected occurrence of [0m[48;2;0;0;64m[97mfnurgle[0m without a headword at line [33m1:0-7[39m of [36m\"../hub/[0m\n[33m[39m[36mtest-files/broken.pf\"[39m."},
		{"2 + 2", "4"},
		{`hub halt "broken"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + hub.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}
