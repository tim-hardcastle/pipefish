package pf_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tim-hardcastle/pipefish/source/pf"
	"github.com/tim-hardcastle/pipefish/source/test_helper"
	"github.com/tim-hardcastle/pipefish/source/text"
)

// We can test the `pf` package by rerunning the tests for the hub, since the hub wraps
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
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestApi(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/foo.pf"`, `Starting script [36m"foo.pf"[39m as service [36m"foo"[39m.`},
		{`hub api`, "\x1b[1m\x1b[3m≡≡≡≡ foo ≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡≡\n\x1b[0m\n\x1b[3m―――― Functions ――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――――\n\x1b[0m\n\x1b[0m  ▪ foo\x1b[38;2;255;215;0m(\x1b[0mx \x1b[38;2;78;201;176many?\x1b[0m\x1b[38;2;255;215;0m)\x1b[0m"},
		{`hub halt "foo"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
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
func TestEnv(t *testing.T) {
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub env "foo"::42`, `OK`},
		{`hub env delete "foo"`, `OK`},
		{`hub env key "", "foo"`, "[32mOK[0m"},
		{`hub env wipe`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
	}
	test_helper.RunHubTest(t, "default", test)
}

func TestBrokenService(t *testing.T) { // We want to make sure that if the service is broken, queries get handed off to the empty service.
	// no t.Parallel()
	test := []test_helper.TestItem{
		{`hub run "../hub/test-files/broken.pf"`, "Starting script [36m\"broken.pf\"[39m as service [36m\"broken\"[39m. [0] [31mError[39m: unexpected occurrence of [0m[48;2;0;0;64m[97mfnurgle[0m without a headword at line [33m1:0-7[39m of [36m\"../hub/[0m\n[33m[39m[36mtest-files/broken.pf\"[39m."},
		{"2 + 2", "4"},
		{`hub halt "broken"`, `OK`},
		{`hub quit`, "[32mOK[0m\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!"},
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
		{`hub values`, "\x1b[31mHub error\x1b[39m: there are no recent errors."},
		{`flibble`, "[0] \x1b[31mError\x1b[39m: identifier \x1b[0m\x1b[48;2;0;0;64m\x1b[97mflibble\x1b[0m is undeclared at line \x1b[33m1:0-7\x1b[39m of REPL input."},
		{`hub values`, "\x1b[31mHub error\x1b[39m: no values were passed."},
		{`"foo"[3]`, "[0] \x1b[31mError\x1b[39m: index \x1b[0m\x1b[48;2;0;0;64m\x1b[97m3\x1b[0m is out of range 0::3 at line \x1b[33m1:5-6\x1b[39m of REPL input. \n\nValues are available with \x1b[0m\x1b[48;2;0;0;64m\x1b[97mhub values\x1b[0m."},
		{`hub values`, "Values passed were:\n\n  ▪ \"foo\"\n  ▪ 3"},
	}
	test_helper.RunHubTest(t, "default", test)
}

type person struct {
	Name string
	Age  int
}

func TestMisc(t *testing.T) {
	// no t.Parallel()
	wd, _ := os.Getwd()
	pfFile, _ := filepath.Abs(filepath.Join(wd, "/../hub/test-files/togo.pf"))
	srv := pf.NewService()
	srv.InitializeFromFilepath(pfFile)
	fortytwo, _ := srv.Do(`42`)
	runeClone, _ := srv.Do(`RuneClone 'q'`)
	person, _ := srv.Do(`Person "Joe", 22`)
	green, _ := srv.Do(`GREEN`)
	if srv.IsClone(fortytwo) {
		t.Fatal("Thinks 42 is clone.")
	}
	if srv.IsStruct(fortytwo) {
		t.Fatal("Thinks 42 is struct.")
	}
	if srv.IsEnum(fortytwo) {
		t.Fatal("Thinks 42 is enum.")
	}
	if srv.IsClone(fortytwo) {
		t.Fatal("Thinks 42 is clone.")
	}
	if srv.IsStruct(fortytwo) {
		t.Fatal("Thinks 42 is struct.")
	}
	if srv.IsEnum(fortytwo) {
		t.Fatal("Thinks 42 is enum.")
	}
	if name, _ := srv.TypeToTypeName(fortytwo.T); name != "int" {
		t.Fatal("TypeToTypename is broken.")
	}
	if name, _ := srv.TypeToTypeName(green.T); name != "Color" {
		t.Fatal("TypeToTypename is broken.")
	}
	if name, _ := srv.TypeToTypeName(person.T); name != "Person" {
		t.Fatal("TypeToTypename is broken.")
	}
	if name, _ := srv.TypeToTypeName(runeClone.T); name != "RuneClone" {
		t.Fatal("TypeToTypename is broken.")
	}
	if !srv.IsClone(runeClone) {
		t.Fatal("Can't recognize clone.")
	}
	if !srv.IsStruct(person) {
		t.Fatal("Can't recognize  struct.")
	}
	if !srv.IsEnum(green) {
		t.Fatal("Can't recognize enum.")
	}
	if srv.UnderlyingType(runeClone) != pf.RUNE {
		t.Fatal("Can't get underlying type.")
	}
	if srv.ToString(green) != "GREEN" {
		t.Fatal("ToString is broken.")
	}
	if srv.ToLiteral(green) != "GREEN" {
		t.Fatal("ToLiteral is broken.")
	}
}

func TestToGo(t *testing.T) {
	// no t.Parallel()
	wd, _ := os.Getwd()
	pfFile, _ := filepath.Abs(filepath.Join(wd, "/../hub/test-files/togo.pf"))
	srv := pf.NewService()
	srv.InitializeFromFilepath(pfFile)
	pfVal, _ := srv.Do(`42`)
	goVal, _ := srv.ToGoWithType(pfVal, reflect.TypeFor[int]())
	if goVal.(int) != 42 {
		t.Fatal("Can't convert 42 to int.")
	}
	goVal, _ = pf.ToGo[int](srv, pfVal)
	if goVal.(int) != 42 {
		t.Fatal("Can't use ToGo.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[any]())
	if goVal.(int) != 42 {
		t.Fatal("Can't convert to any.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[*int]())
	if *(goVal.(*int)) != 42 {
		t.Fatal("Can't convert to pointers.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[int8]())
	if goVal.(int8) != 42 {
		t.Fatal("Can't convert 42 to int8.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[int16]())
	if goVal.(int16) != 42 {
		t.Fatal("Can't convert 42 to int16.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[int32]())
	if goVal.(int32) != 42 {
		t.Fatal("Can't convert 42 to int32.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[int64]())
	if goVal.(int64) != 42 {
		t.Fatal("Can't convert 42 to int64.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[uint]())
	if goVal.(uint) != 42 {
		t.Fatal("Can't convert 42 to uint.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[uint8]())
	if goVal.(uint8) != 42 {
		t.Fatal("Can't convert 42 to uint8.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[uint16]())
	if goVal.(uint16) != 42 {
		t.Fatal("Can't convert 42 to uint16.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[uint32]())
	if goVal.(uint32) != 42 {
		t.Fatal("Can't convert 42 to uint32.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[uint64]())
	if goVal.(uint64) != 42 {
		t.Fatal("Can't convert 42 to uint64.")
	}
	pfVal, _ = srv.Do(`42.0`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[float32]())
	if goVal.(float32) != 42.0 {
		t.Fatal("Can't convert 42.0 to float32.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[float64]())
	if goVal.(float64) != 42.0 {
		t.Fatal("Can't convert 42.0 to float64.")
	}
	pfVal, _ = srv.Do(`"foo"`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[string]())
	if goVal.(string) != "foo" {
		t.Fatal("Can't convert `foo`.")
	}
	pfVal, _ = srv.Do(`true`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[bool]())
	if goVal.(bool) != true {
		t.Fatal("Can't convert true.")
	}
	pfVal, _ = srv.Do(`'q'`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[rune]())
	if goVal.(rune) != 'q' {
		t.Fatal("Can't convert rune.")
	}
	pfVal, _ = srv.Do(`("fee", "fie", "fo", "fum")`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[[]string]())
	if len(goVal.([]string)) != 4 {
		t.Fatal("Can't convert tuple to slice.")
	}
	if (goVal.([]string))[2] != "fo" {
		t.Fatal("Can't convert tuple to slice.")
	}
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[[4]string]())
	if (goVal.([4]string))[2] != "fo" {
		t.Fatal("Can't convert tuple to array.")
	}
	pfVal, _ = srv.Do(`["fee", "fie", "fo", "fum"]`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[[]string]())
	if len(goVal.([]string)) != 4 {
		t.Fatal("Can't convert list to slice.")
	}
	if (goVal.([]string))[2] != "fo" {
		t.Fatal("Can't convert list to slice.")
	}
	pfVal, _ = srv.Do(`map("a"::5, "b"::6)`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[map[string]int]())
	if len(goVal.(map[string]int)) != 2 {
		t.Fatal("Can't convert map.")
	}
	if (goVal.(map[string]int))["a"] != 5 {
		t.Fatal("Can't convert map.")
	}
	pfVal, _ = srv.Do(`set("a", "b", "c")`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[map[string]struct{}]())
	if len(goVal.(map[string]struct{})) != 3 {
		t.Fatal("Can't convert set.")
	}
	if _, ok := (goVal.(map[string]struct{}))["b"]; !ok {
		t.Fatal("Can't convert set.")
	}
	pfVal, _ = srv.Do(`Person("Joe", 22)`)
	goVal, _ = srv.ToGoWithType(pfVal, reflect.TypeFor[person]())
	if (goVal.(person)).Name != "Joe" || (goVal.(person)).Age != 22 {
		t.Fatal("Can't convert struct.")
	}
}
