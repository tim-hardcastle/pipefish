package hub_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/tim-hardcastle/pipefish/source/test_helper"
	"github.com/tim-hardcastle/pipefish/source/text"
)

func TestApi(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/foo.pf"`, `Starting script [36m"foo.pf"[39m as service [36m"foo"[39m.`},
		{`hub api`, "\x1b[1m\x1b[3m≡≡≡≡ foo ≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡\n\x1b[0m\n\x1b[1m════ Functions ════════════════════════════════════════════════════════════════════════════\n\x1b[0m\n\x1b[36m•\x1b[0m foo\x1b[38;2;255;215;0m(\x1b[0mx \x1b[38;2;78;201;176many?\x1b[0m\x1b[38;2;255;215;0m)\x1b[0m"},
		{`hub halt "foo"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestBrokenService(t *testing.T) { // We want to make sure that if the service is broken, queries get handed off to the empty service.
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/broken.pf"`, "Starting script \x1b[36m\"broken.pf\"\x1b[39m as service \x1b[36m\"broken\"\x1b[39m. \n[0] \x1b[31mError\x1b[39m: unexpected occurrence of \x1b[0m\x1b[48;2;0;0;64m\x1b[97mfnurgle\x1b[0m without a headword at line \x1b[33m1:0-7\x1b[39m of \x1b[36m\"../hub/\x1b[0m\n\x1b[33m\x1b[39m\x1b[36mtest-files/broken.pf\"\x1b[39m."},
		{`hub halt "broken"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestCli(t *testing.T) {
	// no t.Parallel()
	if runtime.GOOS == "windows" {
		return
	}
	wd, _ := os.Getwd()
	tmpExe := filepath.Join(t.TempDir(), "pipefish")

	cmd := exec.Command("go", "build", "-o", tmpExe, "../../")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	result, err := exec.Command(tmpExe, "run", filepath.Join(wd, "test-files/cli.pf")).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, result)
	}
	if string(result) != "Hello world!\n" {
		t.Fatal("Expected \"Hello world!\\n\"`; got " + strconv.Quote(string(result)))
	}
}

func TestDump(t *testing.T) { // We want to make sure that if the service is broken, queries get handed off to the empty service.
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/dump.pf"`, `Starting script [36m"dump.pf"[39m as service [36m"dump"[39m.`},
		{`hub dump "big"`, "# Function dump of `big`\n\n## Code dump for function `big` with sig int\n\n@71 : asgm m263 <- m261  // Assign to memory.\n@72 : gtei m262 <- m263 m265  // Int comparison with >=.\n@73 : asgm m266 <- m262  // Assign to memory.\n@74 : qtru m266 @77  // Test true.\n@75 : asgm m268 <- m267  // Assign to memory.\n@76 : jmp @78  // Jump.\n@77 : asgm m268 <- m3  // Assign to memory.\n@78 : qsat m268 @81  // Test not `UNSAT`.\n@79 : asgm m270 <- m268  // Assign to memory.\n@80 : jmp @82  // Jump.\n@81 : asgm m270 <- m269  // Assign to memory.\n@82 : ret  // Return."},
		{`hub dump m "big"`, "# Function dump of `big`\n\n## Code dump for function `big` with sig int\n\n@71 : asgm m263 <- m261  // Assign to memory.\n@72 : gtei m262 <- m263 m265  // Int comparison with >=.\n@73 : asgm m266 <- m262  // Assign to memory.\n@74 : qtru m266 @77  // Test true.\n@75 : asgm m268 <- m267  // Assign to memory.\n@76 : jmp @78  // Jump.\n@77 : asgm m268 <- m3  // Assign to memory.\n@78 : qsat m268 @81  // Test not `UNSAT`.\n@79 : asgm m270 <- m268  // Assign to memory.\n@80 : jmp @82  // Jump.\n@81 : asgm m270 <- m269  // Assign to memory.\n@82 : ret  // Return.\n\n### Memory dump for function `big` with sig int`\n\nm261 : UNDEFINED VALUE::UNDEFINED VALUE!\nm262 : error::\x1b[31mError\x1b[39m: something unexpected has gone wrong at line \x1b[33m4:6-8\x1b[39m of \x1b[36m\"../hub/test-files/dump.pf\"\x1b[39m. \nm263 : UNDEFINED VALUE::UNDEFINED VALUE!\nm264 : BLING::>=\nm265 : int::100\nm266 : UNDEFINED VALUE::UNDEFINED VALUE!\nm267 : string::\"big\"\nm268 : UNDEFINED VALUE::UNDEFINED VALUE!\nm269 : string::\"small\"\nm270 : UNDEFINED VALUE::UNDEFINED VALUE!"},
		{`hub halt "dump"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestEnv(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub env "foo"::42`, `OK`},
		{`hub env key "", "Default key for testing."`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)

	testB := []test_helper.TestItem{
		{`hub delete env "foo"`, `OK`},
		{`hub nuke env`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", testB)
}

func TestErrors(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{"2 +", "[0] [31mError[39m: can't parse end of line as a prefix at line [33m1:3[39m of REPL input."},
		{`hub why 0`, "\x1b[31mError\x1b[39m: can't parse end of line as a prefix. \n\nYou've put end of line in such a position that it looks like you want it to function as a \x1b[0m\nprefix, but it isn't one. \n\n                                                      Error has reference \x1b[0m\x1b[48;2;0;0;64m\x1b[97m\"parse/prefix\"\x1b[0m."},
		{`hub where 0`, "2 +\x1b[31m\n\x1b[0m   \x1b[31m▔\x1b[0m"},
		{`hub errors`, "[0] \x1b[31mError\x1b[39m: can't parse end of line as a prefix at line \x1b[33m1:3\x1b[39m of REPL input."},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestHttp(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub http`, "\x1b[32mOK\x1b[0m"},
		{`hub run "../hub/test-files/server.pf"`, `Starting script [36m"server.pf"[39m as service [36m"server"[39m.`},
		{`hub run "../hub/test-files/client.pf"`, `Starting script [36m"client.pf"[39m as service [36m"client"[39m.`},
		{`twice 2`, "4"},
		{`hub halt "client"`, `OK`},
		{`hub halt "server"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestLog(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/log.pf"`, `Starting script [36m"log.pf"[39m as service [36m"log"[39m.`},
		{`big 6`, `"small"`},
		{`hub log`, "\x1b[0m  ▪ Log at line 8 : Called \x1b[0m\x1b[48;2;0;0;64m\x1b[97mbig\x1b[0m. \n\x1b[0m  ▪ At line 9 we evaluated the condition \x1b[0m\x1b[48;2;0;0;64m\x1b[97mi >= 100\x1b[0m. The condition failed. \n\x1b[0m  ▪ At line 11 we took the \x1b[0m\x1b[48;2;0;0;64m\x1b[97melse\x1b[0m branch, so at line 12 function \x1b[0m\x1b[48;2;0;0;64m\x1b[97mbig\x1b[0m returned \x1b[0m\x1b[48;2;0;0;64m\x1b[97m\"small\"\x1b[0m."},
		{`hub halt "log"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestRbam(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.UserItem{
		{``, ``, `hub config admin "mmadmin", "Norma", "Mortenson", "marilyn@hollywood.org", "password123"`, "You are logged on as \x1b[36mmmadmin\x1b[39m."},
		{`mmadmin`, `password123`, `hub run "../hub/test-files/foo.pf"`, `Starting script [36m"foo.pf"[39m as service [36m"foo"[39m.`},
		{`mmadmin`, `password123`, `hub let "Users" use "foo"`, `OK`},
		{`mmadmin`, `password123`, `hub services`, "The hub is running the following services:\n\n[32m  ▪ [0mService [36m\"foo\"[39m running script [36m\"foo.pf\"[39m."},
		{`mmadmin`, `password123`, `foo 2`, `4`},
		{`mmadmin`, `password123`, `hub sign off`, "\x1b[32mOK\x1b[39m\n\n┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈\n\nThis is an administered hub and you aren't logged on. Please use either \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub register\x1b[0m to \x1b[0m\nregister as a guest; \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub forgot password(username, email string)\x1b[0m to replace your password; \x1b[0m\nor \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub sign on\x1b[0m to sign on if you're trying to use the hub on the terminal it's running on \x1b[0m\nand you're already registered with this hub."},
		{``, ``, `hub register "jdean", "James", "Dean", "rebel@hollywood.org", "password456"`, "You are logged on as \x1b[36mjdean\x1b[39m."},
		{`jdean`, `password456`, `hub services`, "You do not have access to any services."},
		{`jdean`, `password456`, `hub sign off`, "\x1b[32mOK\x1b[39m\n\n┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈\n\nThis is an administered hub and you aren't logged on. Please use either \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub register\x1b[0m to \x1b[0m\nregister as a guest; \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub forgot password(username, email string)\x1b[0m to replace your password; \x1b[0m\nor \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub sign on\x1b[0m to sign on if you're trying to use the hub on the terminal it's running on \x1b[0m\nand you're already registered with this hub."},
		{``, ``, `hub sign on "mmadmin", "password123"`, "You are logged on as \x1b[36mmmadmin\x1b[39m."},
		{`mmadmin`, `password123`, `hub services of user "jdean"`, "The user \x1b[36mjdean\x1b[39m does not have access to any services."},
		{`mmadmin`, `password123`, `hub add "jdean" to "Users"`, "OK"},
		{`mmadmin`, `password123`, `hub groups of user "jdean"`, "The user \x1b[36mjdean\x1b[39m is a member of the following groups: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mGuests \n\x1b[0m  ▪ \x1b[36m\x1b[39mUsers"},
		{`mmadmin`, `password123`, `hub services of user "jdean"`, "The user \x1b[36mjdean\x1b[39m has access to the following services: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mfoo \n\n\x1b[0m\x1b[36m\x1b[39m"},
		{`mmadmin`, `password123`, `hub services of group "Users"`, "The group \x1b[36mUsers\x1b[39m has access to the following services: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mfoo"},
		{`mmadmin`, `password123`, `hub groups of service "foo"`, "The service \x1b[36mfoo\x1b[39m can be accessed by the following groups: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mUsers"},
		{`mmadmin`, `password123`, `hub users of service "foo"`, "The service \x1b[36mfoo\x1b[39m has the following users: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mjdean \n\x1b[0m  ▪ \x1b[36m\x1b[39mmmadmin \n\n\x1b[0m\x1b[36m\x1b[39m"},
		{`mmadmin`, `password123`, `hub services of group "Guests"`, "The group \x1b[36mGuests\x1b[39m has access to no services."},
		{`mmadmin`, `password123`, `hub create group "Superusers"`, "OK"},
		{`mmadmin`, `password123`, `hub users of group "Superusers"`, "The group \x1b[36mSuperusers\x1b[39m has the following owners: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mmmadmin"},
		{`mmadmin`, `password123`, `hub let "jdean" own "Superusers"`, "OK"},
		{`mmadmin`, `password123`, `hub users of group "Superusers"`, "The group \x1b[36mSuperusers\x1b[39m has the following owners: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mjdean \n\x1b[0m  ▪ \x1b[36m\x1b[39mmmadmin"},
		{`mmadmin`, `password123`, `hub unlet "jdean" own "Superusers"`, "OK"},
		{`mmadmin`, `password123`, `hub users of group "Superusers"`, "The group \x1b[36mSuperusers\x1b[39m has the following owners: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mmmadmin \n\n\x1b[0m\x1b[36m\x1b[39mThe group \x1b[36mSuperusers\x1b[39m has the following users: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39m\x1b[36m\x1b[39mjdean"},
		{`mmadmin`, `password123`, `hub users of group "Users"`, "The group \x1b[36mUsers\x1b[39m has the following owners: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39mmmadmin \n\n\x1b[0m\x1b[36m\x1b[39mThe group \x1b[36mUsers\x1b[39m has the following users: \n\n\x1b[0m  ▪ \x1b[36m\x1b[39m\x1b[36m\x1b[39mjdean"},
		{`mmadmin`, `password123`, `hub halt "foo"`, "OK"},
		{`mmadmin`, `password123`, `$ echo "Hello world!"`, `Hello world!`},
		{`mmadmin`, `password123`, `hub change password "password789"`, "OK"},
		{`mmadmin`, `password789`, `hub sign off`, "\x1b[32mOK\x1b[39m\n\n┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈\n\nThis is an administered hub and you aren't logged on. Please use either \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub register\x1b[0m to \x1b[0m\nregister as a guest; \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub forgot password(username, email string)\x1b[0m to replace your password; \x1b[0m\nor \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub sign on\x1b[0m to sign on if you're trying to use the hub on the terminal it's running on \x1b[0m\nand you're already registered with this hub."},
		{``, ``, "hub services", "\x1b[31mHub error\x1b[39m: this is an administered hub and you aren't logged on. Please use either \x1b[0m\x1b[48;2;0;0;64m\x1b[97m\x1b[0m\n\x1b[31m\x1b[39m\x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub register\x1b[0m to register as a guest; \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub forgot password(username, email string)\x1b[0m to \x1b[0m\nreplace your password; or \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub sign on\x1b[0m to sign on if you're trying to use the hub on the \x1b[0m\nterminal it's running on and you're already registered with this hub."},
		{``, ``, `hub sign on "jdean", "password456"`, "You are logged on as \x1b[36mjdean\x1b[39m."},
		{`jdean`, `password456`, `hub services`, "You have access to the following services: \n\n\x1b[0m  ▪ foo \n\n\x1b[0m"},
		{`jdean`, `password456`, `hub groups`, "You are an member of the following groups: \n\n\x1b[0m  ▪ Guests \n\x1b[0m  ▪ Superusers \n\x1b[0m  ▪ Users"},
		{`jdean`, `password456`, `$ echo "Hello world!"`, "\x1b[31mHub error\x1b[39m: Only administrators can use the shell remotely."},
		{`jdean`, `password456`, `hub sign off`, "\x1b[32mOK\x1b[39m\n\n┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈\n\nThis is an administered hub and you aren't logged on. Please use either \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub register\x1b[0m to \x1b[0m\nregister as a guest; \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub forgot password(username, email string)\x1b[0m to replace your password; \x1b[0m\nor \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub sign on\x1b[0m to sign on if you're trying to use the hub on the terminal it's running on \x1b[0m\nand you're already registered with this hub."},
		{``, ``, `$ echo "Hello world!"`, "\x1b[31mHub error\x1b[39m: Only administrators can use the shell remotely."},
		{``, ``, `hub forgot password "jdean", "rebel@hollywood.org"`, "An email with a replacement password has been sent to \x1b[36mrebel@hollywood.org\x1b[39m."},
		{``, ``, `hub register "brando", "Marlon", "Brando", "kurtz@hollywood.org", "password000"`, "You are logged on as \x1b[36mbrando\x1b[39m."},
		{`brando`, `password000`, `hub nuke my account`, "OK"},
		{``, ``, `hub sign on "mmadmin", "password789"`, "You are logged on as \x1b[36mmmadmin\x1b[39m."},
		{`mmadmin`, `password123`, `hub unadd "jdean" to "Users"`, "OK"},
		{`mmadmin`, `password123`, `hub unlet "Users" use "foo"`, `OK`},
		{`mmadmin`, `password123`, `hub uncreate group "Superusers"`, "OK"},
		{`mmadmin`, `password123`, `hub unregister "jdean"`, "OK"},
		{`mmadmin`, `password789`, `hub nuke admin`, "OK"},
		{``, ``, `hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunUserTest(t, "rbam", test)
}

func TestServices(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{"2 + 2", "4"},
		{`hub services`, `No services are running on this hub.`},
		{`hub run "../hub/test-files/foo.pf"`, `Starting script [36m"foo.pf"[39m as service [36m"foo"[39m.`},
		{`hub reset`, "Restarting script \x1b[36m\"../hub/test-files/foo.pf\"\x1b[39m as service \x1b[36m\"foo\"\x1b[39m."},
		{`hub services`, "The hub is running the following services:\n\n[32m  ▪ [0mService [36m\"foo\"[39m running script [36m\"foo.pf\"[39m."},
		{`foo 2`, `4`},
		{`hub run "../hub/test-files/bar.pf"`, `Starting script [36m"bar.pf"[39m as service [36m"bar"[39m.`},
		{`bar 2`, `6`},
		{`hub switch "foo"`, `OK`},
		{`foo 2`, `4`},
		{`hub live off`, `OK`},
		{`hub live on`, `OK`},
		{`hub switch "qux"`, "\x1b[31mHub error\x1b[39m: service \x1b[36mqux\x1b[39m is not initialized."},
		{`hub halt "foo"`, `OK`},
		{`hub halt "bar"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestShell(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`$ echo "Hello!"`, "Hello!"},
		{"$ return 0", "\x1b[32mOK\x1b[0m"},
		{"$ return 42", "\x1b[31mHub error\x1b[39m: exit status 42"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestTrace(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/trace.pf"`, "Starting script [36m\"trace.pf\"[39m as service [36m\"trace\"[39m."},
		{"foo 0", "[0] \x1b[31mError\x1b[39m: division by zero at line \x1b[33m4:7-10\x1b[39m of \x1b[36m\"../hub/test-files/trace.pf\"\x1b[39m."},
		{"hub trace", "\x1b[31mError\x1b[39m: division by zero \nFrom: \x1b[0m\x1b[48;2;0;0;64m\x1b[97mfoo\x1b[0m at line \x1b[33m1:0-3\x1b[39m of REPL input. From: \x1b[0m\x1b[48;2;0;0;64m\x1b[97mdiv\x1b[0m at line \x1b[33m4:7-10\x1b[39m of \x1b[36m\"../hub/test-files/\x1b[0m\n\x1b[33m\x1b[39m\x1b[36mtrace.pf\"\x1b[39m. From: \x1b[0m\x1b[48;2;0;0;64m\x1b[97mdiv\x1b[0m at line \x1b[33m4:7-10\x1b[39m of \x1b[36m\"../hub/test-files/trace.pf\"\x1b[39m."},
		{`hub halt "trace"`, "OK"},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestValues(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/vals.pf"`, "Starting script [36m\"vals.pf\"[39m as service [36m\"vals\"[39m."},
		{`hub values`, "\x1b[31mHub error\x1b[39m: there are no recent errors."},
		{`flibble`, "[0] \x1b[31mError\x1b[39m: identifier \x1b[0m\x1b[48;2;0;0;64m\x1b[97mflibble\x1b[0m is undeclared at line \x1b[33m1:0-7\x1b[39m of REPL input."},
		{`hub values`, "\x1b[31mHub error\x1b[39m: no values were passed."},
		{`"foo"[three]`, "[0] \x1b[31mError\x1b[39m: index \x1b[0m\x1b[48;2;0;0;64m\x1b[97m3\x1b[0m is out of range 0::3 at line \x1b[33m1:5-6\x1b[39m of REPL input. \n\nValues are available with \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub values\x1b[0m."},
		{`hub values`, "Values passed were:\n\n  ▪ \"foo\"\n  ▪ 3"},
		{`hub halt "vals"`, "OK"},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}
