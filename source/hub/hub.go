package hub

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/caddyserver/certmagic"
	"github.com/lmorg/readline/v4"
	"golang.org/x/crypto/pbkdf2"

	"github.com/tim-hardcastle/pipefish/source/database"
	"github.com/tim-hardcastle/pipefish/source/dtypes"
	"github.com/tim-hardcastle/pipefish/source/err"
	"github.com/tim-hardcastle/pipefish/source/pf"
	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/text"
	"github.com/tim-hardcastle/pipefish/source/values"
)

type Hub struct {
	hubFilepath            string
	Services               map[string]*pf.Service // The services the hub knows about.
	ers                    []*pf.Error            // The errors produced by the latest compilation/execution of one of the hub's services.
	Out                    io.Writer
	snap                   *Snap
	oldServiceName         string // Somewhere to keep the old service name while taking a snap. TODO --- you can now take snaps on their own dedicated hub, saving a good deal of faffing around.
	Sources                map[string][]string
	lastRun                []string
	CurrentForm            *Form // TODO!!! --- deprecate, you've had IO for a while.
	Db                     *sql.DB
	administered           bool
	listeningToHttpOrHttps bool
	port                   string
	// The username and password of the person logged into the terminal.
	TerminalUsername string
	TerminalPassword string
	// The usernames and password of whoever called `hub.Do``.
	username, password string
	store              values.Map
	storekey           string
}

func New(path string, out io.Writer) *Hub {
	h := Hub{
		Services: make(map[string]*pf.Service),
		Out:      out,
		lastRun:  []string{},
	}
	h.OpenHubFile(filepath.Join(path, "hub.hub"))
	return &h
}

func (hub *Hub) currentServiceName() string {
	cs := hub.getSV("currentService")
	if cs.T == pf.NULL {
		return ""
	} else {
		return cs.V.(string)
	}
}

func (hub *Hub) hasDatabase() bool {
	return hub.getSV("database").T != pf.NULL
}

func (hub *Hub) getDB() (string, string, string, int, string, string) {
	dbStruct := hub.getSV("database").V.([]pf.Value)
	driver := hub.Services["hub"].ToLiteral(dbStruct[0])
	return driver, dbStruct[1].V.(string), dbStruct[2].V.(string), dbStruct[3].V.(int), dbStruct[4].V.(string), dbStruct[5].V.(string)
}

func (hub *Hub) setDB(driver, name, path string, port int, username, password string) {
	hubService := hub.Services["hub"]
	driverAsEnumValue, _ := hubService.Do(driver)
	structType, _ := hubService.TypeNameToType("Database")
	hub.setSV("database", structType, []pf.Value{driverAsEnumValue, {pf.STRING, name}, {pf.STRING, path}, {pf.INT, port}, {pf.STRING, username}, {pf.STRING, password}})
}

func (hub *Hub) isLive() bool {
	return hub.getSV("isLive").V.(bool)
}

func (hub *Hub) setLive(b bool) {
	hub.setSV("isLive", pf.BOOL, b)
}

func (hub *Hub) setServiceName(name string) {
	hub.setSV("currentService", pf.STRING, name)
}

func (hub *Hub) makeEmptyServiceCurrent() {
	hub.setSV("currentService", pf.NULL, nil)
}

func (hub *Hub) getSV(sv string) pf.Value {
	v, _ := hub.Services["hub"].GetVariable(sv)
	return v
}

func (hub *Hub) setSV(sv string, ty pf.Type, v any) {
	hub.Services["hub"].SetVariable(sv, ty, v)
}

// This converts a string identifying the color of a token (e.g. `type`,
// `number`, to Linux control codes giving the correct coloring according
// to the color theme of the hub.)
func (hub *Hub) getFonts() *values.Map {
	theme := hub.getSV("theme")
	if theme.V == nil {
		return nil
	}
	mapOfThemes := hub.getSV("THEMES").V.(*values.Map)
	mapForTheme, themeExists := mapOfThemes.Get(theme)
	if !themeExists {
		return nil
	}
	fonts := mapForTheme.V.(*values.Map)
	return fonts
}

// This takes the input from the REPL, interprets it as a hub command if it begins with 'hub';
// as an instruction to the os if it begins with 'os', and as an expression to be passed to
// the current service if none of the above hold.
func (hub *Hub) Do(line, username, password, passedServiceName string, external bool) (string, bool) {
	if hub.administered && !hub.listeningToHttpOrHttps && hub.TerminalPassword == "" &&
		!(line == "hub register" || line == "hub log on" || line == "hub quit") {
		hub.WriteError("this is an administered hub and you aren't logged on. Please enter either " +
			"`hub register` to register as a user, or `hub log on` to log on if you're already registered " +
			"with this hub.")
		return passedServiceName, false
	}

	// Otherwise, we're talking to the current service.

	serviceToUse, ok := hub.Services[passedServiceName]
	if !ok {
		hub.WriteError("the hub can't find the service <C>\"" + passedServiceName + "\"</>.")
		return passedServiceName, false
	}

	// We may be talking to the hub itself.

	hubWords := strings.Fields(line)
	if len(hubWords) > 0 && hubWords[0] == "hub" {
		if len(hubWords) == 1 {
			hub.WriteError("you need to say what you want the hub to do.")
			return passedServiceName, false
		}
		hub.username = username
		hub.password = password
		hub.DoHubCommand(strings.Join(hubWords[1:], " "))
		return passedServiceName, false
	}

	// We may be talking to the os

	if len(hubWords) > 0 && hubWords[0] == "os" {
		if hub.isAdministered() {
			hub.WriteError("for reasons of safety and sanity, the `os` prefix doesn't work in administered hubs.")
			return passedServiceName, false
		}
		if len(hubWords) == 3 && hubWords[1] == "cd" { // Because cd changes the directory for the current
			os.Chdir(hubWords[2])     // process, if we did it with exec it would do it for
			hub.WriteString(GREEN_OK) // that process and not for Pipefish.
			return passedServiceName, false
		}
		command := exec.Command(hubWords[1], hubWords[2:]...)
		out, err := command.Output()
		if err != nil {
			hub.WriteError(err.Error())
			return passedServiceName, false
		}
		if len(out) == 0 {
			hub.WriteString(GREEN_OK)
			return passedServiceName, false
		}
		hub.WriteString(string(out))
		return passedServiceName, false
	}

	if hub.currentServiceName() == "#snap" {
		hub.snap.AddInput(line)
	}

	// The service may be broken, in which case we'll let the empty service handle the input.
	if serviceToUse.IsBroken() {
		serviceToUse = hub.Services[""]
	}

	hub.Sources["REPL input"] = []string{line}

	// If we're livecoding we may need to recompile.
	hub.update()
	serviceToUse = hub.Services[hub.currentServiceName()]
	if serviceToUse.IsBroken() {
		return passedServiceName, false
	}

	if match, _ := regexp.MatchString(`^\s*(|\/\/.*)$`, line); match {
		hub.WriteString("")
		return passedServiceName, false
	}

	// *** THIS IS THE BIT WHERE WE DO THE THING!
	val := ServiceDo(serviceToUse, line)
	// *** FROM ALL THAT LOGIC, WE EXTRACT ONE PIPEFISH VALUE !!!
	errorsExist, _ := serviceToUse.ErrorsExist()
	if errorsExist { // Any lex-parse-compile errors should end up in the parser of the compiler of the service, returned in p.
		hub.GetAndReportErrors(serviceToUse)
		return passedServiceName, false
	}

	if val.T == pf.ERROR && !external {
		e := val.V.(*pf.Error)
		if e.Message == "" {
			e = err.CreateErr(e.ErrorId, e.Token, e.Args...)
		}
		hub.WriteString("\n")
		hub.WritePretty("[0] " + text.ERROR + e.Message + text.DescribePos(e.Token))
		hub.WriteString("\n\n")
		hub.ers = []*pf.Error{e}
		if len(e.Values) > 0 {
			hub.WritePretty("Values are available with `hub values`.")
			hub.WriteString("\n\n")
		}
	} else if !serviceToUse.PostHappened() {
		serviceToUse.Output(val)
		if hub.currentServiceName() == "#snap" {
			hub.snap.AddOutput(serviceToUse.ToLiteral(val))
		}
	}
	return passedServiceName, false
}

func (hub *Hub) update() {
	needsUpdate := hub.serviceNeedsUpdate(hub.currentServiceName())
	if hub.isLive() && needsUpdate {
		path, _ := hub.Services[hub.currentServiceName()].GetFilepath()
		hub.StartAndMakeCurrent(hub.TerminalUsername, hub.currentServiceName(), path)
	}
}

func (hub *Hub) DoHubCommand(line string) {
	hubService := hub.Services["hub"]
	hubReturn := ServiceDo(hubService, line)
	if hubReturn.T == pf.OK {
		return
	}
	if errorsExist, _ := hubService.ErrorsExist(); errorsExist { 
		hub.GetAndReportErrors(hubService)
		return
	}
	if hubReturn.T == pf.ERROR {
		hub.WriteError(hubReturn.V.(*pf.Error).Message)
		return
	}
	hub.WriteError("couldn't parse hub instruction.")
	return
}

// Quick and dirty auxilliary function for when we know the value is in fact a string.
func toStr(v pf.Value) string {
	return v.V.(string)
}

type hubWriter struct {
	hub *Hub
}

func (hw hubWriter) Write(b []byte) (int, error) {
	bits := strings.Split(string(b), ", ")
	verb := bits[0]
	args := bits[1:]
	hub := hw.hub
	username := hub.username
	var isAdmin bool
	var err error
	if hub.isAdministered() {
		isAdmin, err = database.IsUserAdmin(hub.Db, username)
		if err != nil {
			hub.WriteError(err.Error())
			return len(b), nil
		}
		if !isAdmin && !greenList.Contains(verb) {
			hub.WriteError("you don't have the admin status necessary to do that.")
			return len(b), nil
		}
		if username == "" && !(verb == "log-on" || verb == "register" || verb == "quit") {
			hub.WriteError("\nthis is an administered hub and you aren't logged on. Please enter either " +
				"`hub register` to register as a guest, or `hub log on` to log on if you're already registered " +
				"with this hub.")
			return len(b), nil
		}
	} else {
		if rbamVerbs.Contains(verb) {
			hub.WriteError("this hub doesn't have RBAM intitialized.")
		}
	}

	switch verb {
	case "add":
		err := database.IsUserGroupOwner(hub.Db, username, args[1])
		if err != nil {
			hub.WriteError(err.Error())
		}
		err = database.AddUserToGroup(hub.Db, args[0], args[1], false)
		if err != nil {
			hub.WriteError(err.Error())
		}
		hub.WriteString(GREEN_OK + "\n")
	case "api":
		hub.update()
		hub.WriteString(hub.Services[hub.currentServiceName()].Api(hub.currentServiceName(), hub.getFonts(), hub.getSV("width").V.(int)))
	case "config-admin":
		if !hub.isAdministered() {
			hub.configAdmin()
		} else {
			hub.WriteError("this hub is already administered.")
		}
	case "config-db":
		hub.configDb()
	case "create":
		err := database.AddGroup(hub.Db, args[0])
		if err != nil {
			hub.WriteError(err.Error())
		}
		err = database.AddUserToGroup(hub.Db, username, args[0], true)
		if err != nil {
			hub.WriteError(err.Error())
		}
		hub.WriteString(GREEN_OK + "\n")
	case "edit":
		command := exec.Command("vim", args[0])
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		err := command.Run()
		if err != nil {
			hub.WriteError(err.Error())
		}
	case "env":
		// $_env has been updated by hub.pf. This is called by both `env` and `env delete`.
		env, _ := hub.Services["hub"].GetVariable("$_env")
		hub.store = *env.V.(*values.Map)
		hub.SaveAndPropagateHubStore()
		hub.WriteString(GREEN_OK + "\n")
	case "env-key":
		if hub.storekey != "" {
			rline := readline.NewInstance()
			rline.SetPrompt("Enter the current environment key for the hub: ")
			rline.PasswordMask = '▪'
			storekey, _ := rline.Readline()
			if storekey != hub.storekey {
				hub.WriteError("incorrect environment key.")
			}
		}
		rline := readline.NewInstance()
		rline.SetPrompt("Enter the new environment key: ")
		rline.PasswordMask = '▪'
		storekey, _ := rline.Readline()
		hub.storekey = storekey
		hub.SaveAndPropagateHubStore()
		hub.WriteString(GREEN_OK + "\n")
	case "env-wipe":
		hub.storekey = ""
		hub.store = values.Map{}
		hub.SaveAndPropagateHubStore()
		hub.WriteString(GREEN_OK + "\n")
	case "errors":
		r, _ := hub.Services[hub.currentServiceName()].GetErrorReport()
		hub.WritePretty(r)
	case "groups-of-user":
		result, err := database.GetGroupsOfUser(hub.Db, args[0], false)
		if err != nil {
			hub.WriteError(err.Error())
		} else {
			hub.WriteString(result)
		}
	case "groups-of-service":
		result, err := database.GetGroupsOfService(hub.Db, args[0])
		if err != nil {
			hub.WriteError(err.Error())
		} else {
			hub.WriteString(result)
		}
	case "halt":
		var name string
		_, ok := hub.Services[args[0]]
		if ok {
			name = args[0]
		} else {
			hub.WriteError("the hub can't find the service <C>\"" + args[0] + "\"</>.")
		}
		if name == "" || name == "hub" {
			hub.WriteError("the hub doesn't know what you want to stop.")
		}
		delete(hub.Services, name)
		hub.WriteString(GREEN_OK + "\n")
		if name == hub.currentServiceName() {
			hub.makeEmptyServiceCurrent()
		}
	case "help":
		if helpMessage, ok := helpStrings[args[0]]; ok {
			hub.WritePretty(helpMessage + "\n")
		} else {
			hub.WriteError("the `hub help` command doesn't accept " +
				"`\"" + args[0] + "\"` as a parameter.")
		}
	case "let":
		isAdmin, err := database.IsUserAdmin(hub.Db, username)
		if err != nil {
			hub.WriteError(err.Error())
		}
		if !isAdmin {
			hub.WriteError("you don't have the admin status necessary to do that.")
		}
		err = database.LetGroupUseService(hub.Db, args[0], args[1])
		if err != nil {
			hub.WriteError(err.Error())
		}
		hub.WriteString(GREEN_OK + "\n")
	case "http":
		hub.WriteString(GREEN_OK)
		go hub.StartHttp([]string{args[0]}, false)
	case "https":
		if len(args) == 0 {
			hub.WriteError("list of domain names cannot be empty")
		}
		hub.WriteString(GREEN_OK)
		domains := []string{}
		for _, arg := range args {
			domains = append(domains, arg)
		}
		go hub.StartHttp(domains, true)
	case "live-on":
		hub.setLive(true)
	case "live-off":
		hub.setLive(false)
	case "log":
		tracking, _ := hub.Services[hub.currentServiceName()].GetTrackingReport()
		hub.WritePretty(tracking)
		hub.WriteString("\n")
	case "log-on":
		hub.getLogin()
	case "log-off":
		hub.TerminalUsername = ""
		hub.TerminalPassword = ""
		hub.makeEmptyServiceCurrent()
		hub.WriteString("\n" + GREEN_OK + "\n")
		hub.WritePretty("\nThis is an administered hub and you aren't logged on. Please enter either " +
			"`hub register` to register as a user, or `hub log on` to log on if you're already registered " +
			"with this hub.\n\n")
	case "groups":
		result, err := database.GetGroupsOfUser(hub.Db, username, true)
		if err != nil {
			hub.WriteError(err.Error())
		} else {
			hub.WriteString(result)
		}
	case "quit":
		hub.Quit()
	case "register":
		hub.addUserAsGuest()
	case "replay":
		hub.oldServiceName = hub.currentServiceName()
		hub.playTest(args[0], false)
		hub.setServiceName(hub.oldServiceName)
		_, ok := hub.Services["#test"]
		if ok {
			delete(hub.Services, "#test")
		}
	case "replay-diff":
		hub.oldServiceName = hub.currentServiceName()
		hub.playTest(args[0], true)
		hub.setServiceName(hub.oldServiceName)
		_, ok := hub.Services["#test"]
		if ok {
			delete(hub.Services, "#test")
		}
	case "reset":
		serviceToReset, ok := hub.Services[hub.currentServiceName()]
		if !ok {
			hub.WriteError("the hub can't find the service <C>\"" + hub.currentServiceName() + "\".")
		}
		if hub.currentServiceName() == "" {
			hub.WriteError("service is empty, nothing to reset.")
		}
		filepath, _ := serviceToReset.GetFilepath()
		hub.WritePretty("Restarting script <C>\"" + filepath +
			"\"</> as service <C>\"" + hub.currentServiceName() + "\"</>.\n")
		hub.StartAndMakeCurrent(username, hub.currentServiceName(), filepath)
		hub.lastRun = []string{hub.currentServiceName()}
	case "rerun":
		if len(hub.lastRun) == 0 {
			hub.WriteError("nothing to rerun.")
		}
		filepath, _ := hub.Services[hub.lastRun[0]].GetFilepath()
		hub.WritePretty("Rerunning script <C>\"" + filepath +
			"</>\" as service <C>\"" + hub.lastRun[0] + "\"</>.\n")
		hub.StartAndMakeCurrent(username, hub.lastRun[0], filepath)
		hub.tryMain()
	case "run":
		fname := args[0]
		sname := args[1]
		if sname == "" {
			sname = text.ExtractFileName(fname)
		}
		if filepath.IsLocal(fname) {
			dir, _ := os.Getwd()
			fname = filepath.Join(dir, fname)
		}
		hub.lastRun = []string{fname, sname}
		hub.WritePretty("Starting script <C>\"" + filepath.Base(fname) + "\"</> as service <C>\"" + sname + "\"</>.\n")
		hub.StartAndMakeCurrent(username, sname, fname)
		hub.tryMain()
	case "serialize":
		hub.WriteString(hub.Services[args[0]].SerializeApi())
	case "services":
		if hub.isAdministered() {
			result, err := database.GetServicesOfUser(hub.Db, username, true)
			if err != nil {
				hub.WriteError(err.Error())
			} else {
				hub.WriteString(result)
			}
		} else {
			if len(hub.Services) == 2 {
				hub.WriteString("The hub isn't running any services.\n")
			}
			hub.WriteString("\n")
			hub.list()
		}
	case "services-of-user":
		result, err := database.GetServicesOfUser(hub.Db, args[0], false)
		if err != nil {
			hub.WriteError(err.Error())
		} else {
			hub.WriteString(result)
		}
	case "services-of-group":
		result, err := database.GetServicesOfGroup(hub.Db, args[0])
		if err != nil {
			hub.WriteError(err.Error())
		} else {
			hub.WriteString(result)
		}
	case "snap":
		scriptFilepath := args[0]
		if filepath.IsLocal(scriptFilepath) {
			dir, _ := os.Getwd()
			scriptFilepath = filepath.Join(dir, scriptFilepath)
		}
		testFilepath := args[1]
		if testFilepath == "" {
			testFilepath = getUnusedTestFilename(scriptFilepath) // If no filename is given, we just generate one.
		}
		hub.snap = NewSnap(scriptFilepath, testFilepath)
		hub.oldServiceName = hub.currentServiceName()
		if hub.StartAndMakeCurrent(username, "#snap", scriptFilepath) {
			snapService := hub.Services["#snap"]
			ty, _ := snapService.TypeNameToType("$_OutputAs")
			snapService.SetVariable("$_outputAs", ty, 0)
			hub.WriteString("Serialization is ON.\n")
			in, out := MakeSnapIo(snapService, hub.Out, hub.snap)
			currentService := snapService
			currentService.SetInHandler(in)
			currentService.SetOutHandler(out)
		} else {
			hub.WriteError("failed to start snap")
		}
	case "snap-good":
		if hub.currentServiceName() != "#snap" {
			hub.WriteError("you aren't taking a snap.")
		}
		result := hub.snap.Save(GOOD)
		hub.WriteString(result + "\n")
		hub.setServiceName(hub.oldServiceName)
	case "snap-bad":
		if hub.currentServiceName() != "#snap" {
			hub.WriteError("you aren't taking a snap.")
		}
		result := hub.snap.Save(BAD)
		hub.WriteString(result + "\n")
		hub.setServiceName(hub.oldServiceName)
	case "snap-record":
		if hub.currentServiceName() != "#snap" {
			hub.WriteError("you aren't taking a snap.")
		}
		result := hub.snap.Save(RECORD)
		hub.WriteString(result + "\n")
		hub.setServiceName(hub.oldServiceName)
	case "snap-discard":
		if hub.currentServiceName() != "#snap" {
			hub.WriteError("you aren't taking a snap.")
		}
		hub.WriteString(GREEN_OK + "\n")
		hub.setServiceName(hub.oldServiceName)
	case "switch":
		sname := args[0]
		_, ok := hub.Services[sname]
		if ok {
			hub.WriteString(GREEN_OK + "\n")
			if hub.administered {
				access, err := database.DoesUserHaveAccess(hub.Db, username, sname)
				if err != nil {
					hub.WriteError("o/ " + err.Error())
				}
				if !access {
					hub.WriteError("you don't have access to service <C>\"" + sname + "\"</>.")
				}
				database.UpdateService(hub.Db, username, sname)
			} else {
				hub.setServiceName(sname)
			}
			break
		}
		hub.WriteError("service <C>\"" + sname + "\"</> doesn't exist.")
	case "test":
		fname := args[0]
		if filepath.IsLocal(fname) {
			dir, _ := os.Getwd()
			fname = filepath.Join(dir, fname)
		}
		file, err := os.Open(fname)
		if err != nil {
			hub.WriteError(strings.TrimSpace(err.Error()) + "\n")
			break
		}
		defer file.Close()
		fileInfo, err := file.Stat()
		if err != nil {
			hub.WriteError(strings.TrimSpace(err.Error()) + "\n")
			break
		}
		if fileInfo.IsDir() {
			files, err := file.Readdir(0)
			if err != nil {
				hub.WriteError(strings.TrimSpace(err.Error()) + "\n")
				break
			}
			for _, potentialPfFile := range files {
				if filepath.Ext(potentialPfFile.Name()) == ".pf" {
					hub.TestScript(fname+"/"+potentialPfFile.Name(), ERROR_CHECK)
				}
			}
		} else {
			hub.TestScript(fname, ERROR_CHECK)
		}
	case "trace":
		if len(hub.ers) == 0 {
			hub.WriteError("there are no recent errors.")
			break
		}
		if len(hub.ers[0].Trace) == 0 {
			hub.WriteError("not a runtime error.")
			break
		}
		hub.WritePretty(pf.GetTraceReport(hub.ers[0]))
	case "users-of-group":
		result, err := database.GetUsersOfGroup(hub.Db, args[0])
		if err != nil {
			hub.WriteError(err.Error())
		} else {
			hub.WriteString(result)
		}
	case "users-of-service":
		result, err := database.GetUsersOfService(hub.Db, args[0])
		if err != nil {
			hub.WriteError(err.Error())
		} else {
			hub.WriteString(result)
		}
	case "values":
		if len(hub.ers) == 0 {
			hub.WriteError("there are no recent errors.")
			break
		}
		if hub.ers[0].Values == nil {
			hub.WriteError("no values were passed.")
			break
		}
		if len(hub.ers[0].Values) == 0 {
			hub.WriteError("no values were passed.")
			break
		}
		if len(hub.ers[0].Values) == 1 {
			hub.WriteString("\nThe value passed was:\n\n")
		} else {
			hub.WriteString("\nValues passed were:\n\n")
		}
		for _, v := range hub.ers[0].Values {
			if v.T == pf.BLING {
				hub.WriteString(BULLET_SPACING + hub.Services[hub.currentServiceName()].ToLiteral(v))
			} else {
				hub.WriteString(BULLET + hub.Services[hub.currentServiceName()].ToLiteral(v))
			}
			hub.WriteString("\n")
		}
		hub.WriteString("\n")
	case "where":
		num, _ := strconv.Atoi(args[0])
		if num < 0 {
			hub.WriteError("the `where` keyword can't take a negative number as a parameter.")
			break
		}
		if num >= len(hub.ers) {
			hub.WriteError("there aren't that many errors.")
			break
		}
		println()
		line := hub.Sources[hub.ers[num].Token.Source][hub.ers[num].Token.Line-1] + "\n"
		startUnderline := hub.ers[num].Token.ChStart
		lenUnderline := hub.ers[num].Token.ChEnd - startUnderline
		if lenUnderline == 0 {
			lenUnderline = 1
		}
		endUnderline := startUnderline + lenUnderline
		hub.WriteString(line[0:startUnderline])
		hub.WriteString(Red(line[startUnderline:endUnderline]))
		hub.WriteString(line[endUnderline:])
		hub.WriteString(strings.Repeat(" ", startUnderline))
		hub.WriteString(Red(strings.Repeat("▔", lenUnderline)))
	case "why":
		hub.WriteString("\n")
		num, _ := strconv.Atoi(args[0])
		if num >= len(hub.ers) {
			hub.WriteError("there aren't that many errors.")
			break
		}
		exp, _ := pf.ExplainError(hub.ers, num)
		hub.WritePretty("<R>Error</>: " + hub.ers[num].Message + ".")
		hub.WriteString("\n\n")
		hub.WritePretty(exp + "\n\n")
		refLine := hub.GetPretty("Error has reference `\"" + hub.ers[num].ErrorId + "\"`.")
		padding := strings.Repeat(" ", hub.getSV("width").V.(int)-len(text.StripColors(refLine))-2)
		hub.WriteString(padding)
		hub.WritePretty(refLine)
		hub.WriteString("\n")
	}
	return len(b), nil
}

func (hub *Hub) makeWriter() io.Writer {
	return hubWriter{
		hub: hub,
	}
}

// Things that only make sense if we have RBAM set up.
var rbamVerbs = dtypes.MakeFromSlice([]string{"add", "create", "log-on", "log-off", "let", 
"register", "groups", "groups-of-user", "groups-of-service", "services of group", 
"services-of-user", "users-of-service", "users-of-group", "let-use", "let-own"})

// Things you can use if you're logged in to a service with RBAM, but not as admin.
var greenList = dtypes.MakeFromSlice([]string{"api", "errors", "help", "log-on", "log-off", 
"groups", "register", "service", "switch", "values", "why", "quit"})

func getUnusedTestFilename(scriptFilepath string) string {
	fname := filepath.Base(scriptFilepath)
	fname = fname[:len(fname)-len(filepath.Ext(fname))]
	dname := filepath.Dir(scriptFilepath)
	directoryName := dname + "/-tests/" + fname
	name := FlattenedFilename(scriptFilepath) + "_"

	tryNumber := 1
	tryName := ""

	for ; ; tryNumber++ {
		tryName = name + strconv.Itoa(tryNumber) + ".tst"
		_, error := os.Stat(directoryName + "/" + tryName)
		if os.IsNotExist(error) {
			break
		}
	}
	return tryName
}

func (hub *Hub) Quit() {
	hub.saveHubFile()
	hub.WriteString(GREEN_OK + "\n" + Logo() + "Thank you for using Pipefish. Have a nice day!\n\n")
	if !testing.Testing() {
		os.Exit(0)
	}
}

func (hub *Hub) help() {
	hub.WriteString("\n")
	hub.WriteString("Help topics are:\n")
	hub.WriteString("\n")
	for _, v := range helpTopics {
		hub.WriteString("  " + BULLET + v + "\n")
	}
	hub.WriteString("\n")
}

func (hub *Hub) WritePretty(s string) {
	// This shouldn't be happening here.
	hubService, ok := hub.Services["hub"]
	if !ok {
		panic("Hub failed to initialize, error is `" + s + "`.")
	}
	mdFunc := hubService.GetMarkdowner("", hub.getSV("width").V.(int), hub.getFonts())
	hub.WriteString(mdFunc(s))
}

func (hub *Hub) GetPretty(s string) string {
	hubService, _ := hub.Services["hub"]
	mdFunc := hubService.GetMarkdowner("", hub.getSV("width").V.(int), hub.getFonts())
	return mdFunc(s)
}

func (hub *Hub) isAdministered() bool {
	_, err := os.Stat(filepath.Join(settings.PipefishHomeDirectory, "user/admin.dat"))
	return !errors.Is(err, os.ErrNotExist)
}

func (hub *Hub) WriteError(s string) {
	hub.WriteString("\n")
	hub.WritePretty(HUB_ERROR + s)
	hub.WriteString("\n\n")
}

func (hub *Hub) WriteString(s string) {
	io.WriteString(hub.Out, s)
}

var helpStrings = map[string]string{}

var helpTopics = []string{}

func init() {
	helpStrings = map[string]string{}
}

func (hub *Hub) StartAndMakeCurrent(username, serviceName, scriptFilepath string) bool {
	if hub.administered {
		err := database.UpdateService(hub.Db, username, serviceName)
		if err != nil {
			hub.WriteError("u/ " + err.Error())
			return false
		}
	}
	hub.setServiceName(serviceName)
	hub.createService(serviceName, scriptFilepath)
	return true
}

func (hub *Hub) tryMain() { // Guardedly tries to run the `main` command.
	if !hub.Services[hub.currentServiceName()].IsBroken() {
		val, _ := hub.Services[hub.currentServiceName()].CallMain()
		hub.lastRun = []string{hub.currentServiceName()}
		switch val.T {
		case pf.ERROR:
			hub.WritePretty("\n[0] " + valToString(hub.Services[hub.currentServiceName()], val))
			hub.WriteString("\n")
			hub.ers = []*pf.Error{val.V.(*pf.Error)}
		case pf.UNDEFINED_TYPE: // Which is what we get back if there is no `main` command.
		default:
			hub.WriteString(valToString(hub.Services[hub.currentServiceName()], val))
		}
	}
}

func (hub *Hub) serviceNeedsUpdate(name string) bool {
	serviceToUpdate, present := hub.Services[name]
	if !present {
		return true
	}
	if name == "" {
		return false
	}
	needsUpdate, err := serviceToUpdate.NeedsUpdate()
	if err != nil {
		hub.WriteError(err.Error())
		return false
	}
	return needsUpdate
}

func (hub *Hub) createService(name, scriptFilepath string) bool {
	needsRebuild := hub.serviceNeedsUpdate(name)
	if !needsRebuild {
		return false
	}
	newService := pf.NewService()
	newService.SetLocalExternalServices(hub.Services)
	if text.Head(scriptFilepath, "!") {
		scriptFilepath = filepath.Join(settings.PipefishHomeDirectory, scriptFilepath[1:])
	}
	e := newService.InitializeFromFilepathWithStore(scriptFilepath, &hub.store) // We get an error only if it completely fails to open the file, otherwise there'll be errors in the Common Parser Bindle as usual.
	hub.Sources, _ = newService.GetSources()
	if newService.IsBroken() {
		if name == "hub" {
			println("Filepath is", scriptFilepath)
			println("Pipefish: unable to compile hub: " + newService.GetErrors()[0].ErrorId + ".")
			println(newService.GetErrors()[0].Message)
			panic("That's all folks!")
		}
		if !newService.IsInitialized() {
			hub.WriteError("unable to open <C>\"" + scriptFilepath + "\"</> with error `" + e.Error() + "`")
			hub.Sources = map[string][]string{}
			hub.makeEmptyServiceCurrent()
		} else {
			hub.Services[name] = newService
			hub.GetAndReportErrors(newService)
		}
		if name == "hub" {
			os.Exit(2)
		}
		return false
	}
	if testing.Testing() {
		newService.SetOutHandler(newService.MakeLiteralOutHandler(hub.Out))
	}
	hub.Services[name] = newService
	return true
}

func StartServiceFromCli() {
	filename := os.Args[2]
	newService := pf.NewService()
	// This ought to get the `$_env` settings.
	// Then we could do proper markdown in the errors.
	newService.InitializeFromFilepathWithStore(filename, &values.Map{})
	if newService.IsBroken() {
		fmt.Println("\nThere were errors running the script <C>\"" + filename + "\"</>.")
		s, _ := newService.GetErrorReport()
		fmt.Println(s)
		fmt.Print("Closing Pipefish.\n\n")
		os.Exit(3)
	}
	val, _ := newService.CallMain()
	if val.T == pf.UNDEFINED_TYPE {
		s := "\nScript <C>\"" + filename + "\"</> has no `main` command.\n\n"
		fmt.Println(s)
		fmt.Print("\n\nClosing Pipefish.\n\n")
		os.Exit(4)
	}
	fmt.Println(newService.ToString(val) + "\n")
	os.Exit(0)
}

func (hub *Hub) GetAndReportErrors(sv *pf.Service) {
	hub.ers = sv.GetErrors()
	r, _ := sv.GetErrorReport()
	hub.WritePretty(r)
}

func (hub *Hub) CurrentServiceIsBroken() bool {
	return hub.Services[hub.currentServiceName()].IsBroken()
}

var prefix = `var

`

func (hub *Hub) saveHubFile() string {
	hubService := hub.Services["hub"]
	var buf strings.Builder
	buf.WriteString(prefix)
	buf.WriteString("allServices = map(")
	serviceList := []string{}
	for k := range hub.Services {
		if k != "" && k[0] != '#' {
			serviceList = append(serviceList, k)
		}
	}
	for i, v := range serviceList {
		buf.WriteString("`")
		buf.WriteString(v)
		buf.WriteString("`::`")
		name, _ := hub.Services[v].GetFilepath()
		buf.WriteString(name)
		buf.WriteString("`")
		if i < len(serviceList)-1 {
			buf.WriteString(",\n               .. ")
		}
	}
	buf.WriteString(")\n\n")
	buf.WriteString("currentService string? = ")
	csV := hub.getSV("currentService")
	if csV.T == values.NULL {
		buf.WriteString("NULL")
	} else {
		cs := csV.V.(string)
		if len(cs) == 0 || cs[0] == '#' {
			buf.WriteString("NULL")
		} else {
			buf.WriteString("`")
			buf.WriteString(cs)
			buf.WriteString("`")
		}
	}
	buf.WriteString("\n\n")
	buf.WriteString("isLive = ")
	buf.WriteString(hubService.ToLiteral(hub.getSV("isLive")))
	buf.WriteString("\n\n")
	buf.WriteString("theme Theme? = ")
	buf.WriteString(hubService.ToLiteral(hub.getSV("theme")))
	buf.WriteString("\n\n")
	buf.WriteString("width = ")
	buf.WriteString(hubService.ToLiteral(hub.getSV("width")))
	buf.WriteString("\n\n")
	buf.WriteString("database Database? = ")
	dbVal := hub.getSV("database")
	if dbVal.T == pf.NULL {
		buf.WriteString("NULL\n")
	} else {
		args := dbVal.V.([]pf.Value)
		buf.WriteString("Database with (driver::")
		buf.WriteString(hubService.ToLiteral(args[0]))
		buf.WriteString(",\n")
		buf.WriteString("                                 .. name::")
		buf.WriteString(hubService.ToLiteral(args[1]))
		buf.WriteString(",\n")
		buf.WriteString("                                 .. host::")
		buf.WriteString(hubService.ToLiteral(args[2]))
		buf.WriteString(",\n")
		buf.WriteString("                                 .. port::")
		buf.WriteString(hubService.ToLiteral(args[3]))
		buf.WriteString(",\n")
		buf.WriteString("                                 .. username::")
		buf.WriteString(hubService.ToLiteral(args[4]))
		buf.WriteString(",\n")
		buf.WriteString("                                 .. password::")
		buf.WriteString(hubService.ToLiteral(args[5]))
		buf.WriteString(")\n")
	}

	fname := hub.MakeFilepath(hub.hubFilepath)

	f, err := os.Create(fname)
	if err != nil {
		return HUB_ERROR + "os reports \"" + strings.TrimSpace(err.Error()) + "\".\n"
	}
	defer f.Close()
	f.WriteString(buf.String())
	return GREEN_OK

}

func (h *Hub) OpenHubFile(hubFilepath string) {
	h.createService("", "")
	h.createService("hub", hubFilepath)
	storePath := hubFilepath[0:len(hubFilepath)-len(filepath.Ext(hubFilepath))] + ".env"
	_, err := os.Stat(storePath)
	if err == nil {
		file, err := os.Open(storePath)
		if err != nil {
			panic("Can't open hub `$_env` data.")
		}
		b, err := io.ReadAll(file)
		if err != nil {
			panic("Can't open hub `$_env` data.")
		}
		s := string(b)
		for ; s != "" && !text.Head(s, "PLAINTEXT"); h.WritePretty("Invalid `env` key. Enter a valid one or press return to continue without loading the store.") {
			salt := s[0:32]
			ciphertext := s[32:]
			rline := readline.NewInstance()
			rline.SetPrompt("Enter the env key for the hub: ")
			rline.PasswordMask = '▪'
			storekey, _ := rline.Readline()
			if storekey == "" {
				println("Starting hub without opening env data.")
				s = "PLAINTEXT"
				break
			}
			key := pbkdf2.Key([]byte(storekey), []byte(salt), 65536, 32, sha256.New) // sha256 has nothing to do with it but the API is stupid.
			block, err := aes.NewCipher(key)
			if err != nil {
				panic(err)
			}
			iv := ciphertext[:aes.BlockSize]
			ciphertext = ciphertext[aes.BlockSize:]
			mode := cipher.NewCBCDecrypter(block, []byte(iv))
			decrypt := make([]byte, len(ciphertext))
			mode.CryptBlocks(decrypt, []byte(ciphertext))
			if string(decrypt[0:9]) == "PLAINTEXT" {
				s = string(decrypt)
				h.storekey = storekey
				break
			}
		}
		bits := strings.Split(strings.TrimSpace(s), "\n")[1:]
		for _, bit := range bits {
			pair, _ := h.Services["hub"].Do(bit)
			h.store = *h.store.Set(pair.V.([]pf.Value)[0], pair.V.([]pf.Value)[1])
		}
	}
	hubService := h.Services["hub"]
	h.hubFilepath = h.MakeFilepath(hubFilepath)
	v, _ := hubService.GetVariable("allServices")
	services := v.V.(pf.Map).AsSlice()

	var driver, name, host, username, password string
	var port int

	if h.hasDatabase() {
		driver, name, host, port, username, password = h.getDB()
		h.Db, _ = database.GetdB(driver, host, name, port, username, password)
	}

	errors := false
	for _, pair := range services {
		serviceName := pair.Key.V.(string)
		serviceFilepath := pair.Val.V.(string)
		if serviceName == "" || serviceName == "hub" {
			continue
		}
		h.createService(serviceName, serviceFilepath)
		errorsExist, _ := h.Services[serviceName].ErrorsExist()
		if errorsExist {
			errors = true
		}
	}
	if errors {
		h.WriteString("\n\n")
	}
	hubService = h.Services["hub"] // TODO
	ty, _ := hubService.TypeNameToType("Hub")
	hubService.SetVariable("HUB", ty, h.makeWriter())
	h.list()
}

func (hub *Hub) SaveAndPropagateHubStore() {
	for _, srv := range hub.Services {
		srv.SetEnv(&hub.store)
	}
	storePath := hub.hubFilepath[0:len(hub.hubFilepath)-len(filepath.Ext(hub.hubFilepath))] + ".env"
	storeDump := hub.Services["hub"].WriteSecret(hub.store, hub.storekey)
	file, _ := os.Create(storePath)
	file.WriteString(storeDump)
}

func (hub *Hub) list() {
	if len(hub.Services) == 2 {
		return
	}
	hub.WriteString("The hub is running the following services:\n\n")
	for k := range hub.Services {
		if k == "" || k == "hub" {
			continue
		}
		fpath, _ := hub.Services[k].GetFilepath()
		if hub.Services[k].IsBroken() {
			hub.WriteString(BROKEN)
			hub.WritePretty("Service <C>\"" + k + "\"</> running script <C>\"" + filepath.Base(fpath) + "\"</>.")
		} else {
			hub.WriteString(GOOD_BULLET)
			hub.WritePretty("Service <C>\"" + k + "\"</> running script <C>\"" + filepath.Base(fpath) + "\"</>.")
		}
		hub.WriteString("\n")
	}
	hub.WriteString("\n")
}

func (hub *Hub) TestScript(scriptFilepath string, testOutputType TestOutputType) {

	fname := filepath.Base(scriptFilepath)
	fname = fname[:len(fname)-len(filepath.Ext(fname))]
	dname := filepath.Dir(scriptFilepath)
	directoryName := dname + "/-tests/" + fname

	hub.oldServiceName = hub.currentServiceName()
	files, _ := os.ReadDir(directoryName)
	for _, testFileInfo := range files {
		testFilepath := directoryName + "/" + testFileInfo.Name()
		hub.RunTest(scriptFilepath, testFilepath, testOutputType)
	}
	_, ok := hub.Services["#test"]
	if ok {
		delete(hub.Services, "#test")
	}
	hub.setServiceName(hub.oldServiceName)

}

func (hub *Hub) RunTest(scriptFilepath, testFilepath string, testOutputType TestOutputType) {

	f, err := os.Open(testFilepath)
	if err != nil {
		hub.WriteError(strings.TrimSpace(err.Error()) + "\n")
		return
	}

	scanner := bufio.NewScanner(f)
	scanner.Scan()
	testType := strings.Split(scanner.Text(), ": ")[1]
	if testType == RECORD {
		f.Close() // TODO --- shouldn't this do something?
		return
	}
	scanner.Scan()
	if !hub.StartAndMakeCurrent("", "#test", scriptFilepath) {
		hub.WriteError("Can't initialize script <C>\"" + scriptFilepath + "\"</>")
		return
	}
	testService := hub.Services["#test"]
	in, out := MakeTestIoHandler(testService, hub.Out, scanner, testOutputType)
	testService.SetInHandler(in)
	testService.SetOutHandler(out)
	if testOutputType == ERROR_CHECK {
		hub.WritePretty("Running test <C>\"" + testFilepath + "\"</>.\n")
	}
	ty, _ := testService.TypeNameToType("$_OutputAs")
	testService.SetVariable("$_outputAs", ty, 0)
	_ = scanner.Scan() // eats the newline
	executionMatchesTest := true
	for scanner.Scan() {
		lineIn := scanner.Text()[3:]
		if testOutputType == SHOW_ALL {
			hub.WriteString("-> " + lineIn + "\n")
		}
		result := ServiceDo(testService, lineIn)
		if errorsExist, _ := testService.ErrorsExist(); errorsExist {
			report, _ := testService.GetErrorReport()
			hub.WritePretty(report)
			f.Close()
			continue
		}
		scanner.Scan()
		lineOut := scanner.Text()
		if valToString(testService, result) != lineOut {
			executionMatchesTest = false
			if testOutputType == SHOW_DIFF {
				hub.WriteString("-> " + lineIn + "\n" + WAS + lineOut + "\n" + GOT + valToString(testService, result) + "\n")
			}
			if testOutputType == SHOW_ALL {
				hub.WriteString(WAS + lineOut + "\n" + GOT + valToString(testService, result) + "\n")
			}
		} else {
			if testOutputType == SHOW_ALL {
				hub.WriteString(lineOut + "\n")
			}
		}
	}
	if testOutputType == ERROR_CHECK {
		if executionMatchesTest && testType == BAD {
			hub.WriteError("bad behavior reproduced by test" + "\n")
			f.Close()
			hub.RunTest(scriptFilepath, testFilepath, SHOW_ALL)
			return
		}
		if !executionMatchesTest && testType == GOOD {
			hub.WriteError("good behavior not reproduced by test" + "\n")
			f.Close()
			hub.RunTest(scriptFilepath, testFilepath, SHOW_ALL)
			return
		}
		hub.WriteString(TEST_PASSED)
	}
	f.Close()
}

func (hub *Hub) playTest(testFilepath string, diffOn bool) {
	f, err := os.Open(testFilepath)
	if err != nil {
		hub.WriteError(strings.TrimSpace(err.Error()) + "\n")
		return
	}
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	_ = scanner.Text() // test type doesn't matter
	scanner.Scan()
	scriptFilepath := (scanner.Text())[8:]
	scanner.Scan()
	hub.StartAndMakeCurrent("", "#test", scriptFilepath)
	testService := (*hub).Services["#test"]
	ty, _ := testService.TypeNameToType("$_OutputAs")
	testService.SetVariable("$_outputAs", ty, 0)
	in, out := MakeTestIoHandler(testService, hub.Out, scanner, SHOW_ALL)
	testService.SetInHandler(in)
	testService.SetOutHandler(out)
	_ = scanner.Scan() // eats the newline
	for scanner.Scan() {
		lineIn := scanner.Text()[3:]
		scanner.Scan()
		lineOut := scanner.Text()
		result := ServiceDo(testService, lineIn)
		if errorsExist, _ := testService.ErrorsExist(); errorsExist {
			report, _ := testService.GetErrorReport()
			hub.WritePretty(report)
			f.Close()
			return
		}
		hub.WriteString("#test → " + lineIn + "\n")

		if valToString(testService, result) == lineOut || !diffOn {
			hub.WriteString(valToString(testService, result) + "\n")
		} else {
			hub.WriteString(WAS + lineOut + "\n" + GOT + valToString(testService, result) + "\n")
		}
	}
}

func valToString(srv *pf.Service, val pf.Value) string {
	// TODO --- the exact behavior of this function should depend on service variables but I haven't put them in the VM yet.
	// Alternately we can leave it as it is and have the vm's Describe method take care of it.
	return srv.ToLiteral(val)
}

func (h *Hub) StartHttp(args []string, isHttps bool) {
	// TODO --- everything that depends on this should depend on something else.
	h.listeningToHttpOrHttps = true
	var err error
	http.HandleFunc("/", h.handleJsonRequest)
	if isHttps {
		handler := http.NewServeMux()
		err = certmagic.HTTPS(args, handler)
	} else {
		err = http.ListenAndServe(":"+args[0], nil)
	}
	if errors.Is(err, http.ErrServerClosed) {
		h.WriteError("server closed.")
	} else { // err is always non-nil.
		h.WriteError("error starting server: " + err.Error())
		return
	}
}

// The hub expects an HTTP request to consist of JSON containing the line to be executed,
// the service to execute it, and the username and password of the user.
type jsonRequest = struct {
	Body     string
	Service  string
	Username string
	Password string
}

type jsonResponse = struct {
	Body    string
	Service string
}

func (h *Hub) handleJsonRequest(w http.ResponseWriter, r *http.Request) {
	var request jsonRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var serviceName string
	if h.administered && !((!h.listeningToHttpOrHttps) && (request.Body == "hub register" || request.Body == "hub log in")) {
		_, err = database.ValidateUser(h.Db, request.Username, request.Password)
		if err != nil {
			h.WriteError(err.Error())
			return
		}
	}
	var buf bytes.Buffer
	h.Out = &buf
	sv := h.Services[request.Service]
	sv.SetOutHandler(sv.MakeLiteralOutHandler(&buf))
	serviceName, _ = h.Do(request.Body, request.Username, request.Password, request.Service, true)
	h.Out = os.Stdout
	response := jsonResponse{Body: buf.String(), Service: serviceName}
	json.NewEncoder(w).Encode(response)
}

// So, the Form type. Yes, I basically am reinventing the object here because the fields of
// a struct aren't first-class objects in Go, unlike other superior langages I could name.
// I can get rid of the whole thing when I do SQL integration and can just make the hub into
// a regular Pipefish service. TODO --- you can do this now!
type Form struct { // For when the hub wants to initiate structured input.
	Fields []string
	Result map[string]string
	Call   func(f *Form)
}

func (h *Hub) addUserAsGuest() {
	h.CurrentForm = &Form{Fields: []string{"Username", "First name", "Last name", "Email", "*Password", "*Confirm password"},
		Call:   func(f *Form) { h.handleConfigUserForm(f) },
		Result: make(map[string]string)}
}

func (h *Hub) handleConfigUserForm(f *Form) {
	h.CurrentForm = nil
	_, err := os.Stat(filepath.Join(settings.PipefishHomeDirectory, "user/admin.dat"))
	if errors.Is(err, os.ErrNotExist) {
		h.WriteError("this Pipefish hub doesn't have administered " +
			"access: there is nothing to join.")
		return
	}
	if err != nil {
		h.WriteError("E/ " + err.Error())
		return
	}
	if f.Result["*Password"] != f.Result["*Confirm password"] {
		h.WriteError("passwords don't match.")
		return
	}

	err = database.AddUser(h.Db, f.Result["Username"], f.Result["First name"],
		f.Result["Last name"], f.Result["Email"], f.Result["*Password"], "")
	if err != nil {
		h.WriteError("F/ " + err.Error())
		return
	}
	err = database.AddUserToGroup(h.Db, f.Result["Username"], "Guests", false)
	if err != nil {
		h.WriteError("G/ " + err.Error())
		return
	}
	h.TerminalUsername = f.Result["Username"]
	h.TerminalPassword = f.Result["*Password"]
	h.WriteString("You are logged in as " + h.TerminalUsername + ".\n")
}

func (h *Hub) configAdmin() {
	h.CurrentForm = &Form{Fields: []string{"Username", "First name", "Last name", "Email", "*Password", "*Confirm password"},
		Call:   func(f *Form) { h.handleConfigAdminForm(f) },
		Result: make(map[string]string)}
}

func (h *Hub) handleConfigAdminForm(f *Form) {
	h.CurrentForm = nil
	if h.Db == nil {
		h.WriteError("database has not been configured: edit the hub file or do `hub config db` first.")
		return
	}
	if f.Result["*Password"] != f.Result["*Confirm password"] {
		h.WriteError("passwords don't match.")
		return
	}
	err := database.AddAdmin(h.Db, f.Result["Username"], f.Result["First name"],
		f.Result["Last name"], f.Result["Email"], f.Result["*Password"], h.currentServiceName(), settings.PipefishHomeDirectory)
	if err != nil {
		h.WriteError("H/ " + err.Error())
		return
	}
	h.WriteString(GREEN_OK + "\n")
	h.TerminalUsername = f.Result["Username"]
	h.TerminalPassword = f.Result["*Password"]
	h.WritePretty("You are logged in as " + h.TerminalUsername + ".\n")

	h.administered = true
}

func (h *Hub) getLogin() {
	h.CurrentForm = &Form{Fields: []string{"Username", "*Password"},
		Call:   func(f *Form) { h.handleLoginForm(f) },
		Result: make(map[string]string)}
}

func (h *Hub) handleLoginForm(f *Form) {
	h.CurrentForm = nil
	_, err := database.ValidateUser(h.Db, f.Result["Username"], f.Result["*Password"])
	if err != nil {
		h.WriteError("I/ " + err.Error())
		h.WriteString("Please try again.\n\n")
		return
	}
	h.TerminalUsername = f.Result["Username"]
	h.TerminalPassword = f.Result["*Password"]
	h.WriteString(GREEN_OK + "\n")
}

func (h *Hub) configDb() {
	h.CurrentForm = &Form{Fields: []string{database.GetDriverOptions(), "Host", "Port", "Database name", "Username for database access", "*Password for database access"},
		Call:   func(f *Form) { h.handleConfigDbForm(f) },
		Result: make(map[string]string)}
}

func (h *Hub) handleConfigDbForm(f *Form) {
	h.CurrentForm = nil
	number, err := strconv.Atoi(f.Result[database.GetDriverOptions()])
	if err != nil {
		h.WriteError("hub/db/config/a: " + err.Error())
		return
	}
	port, err := strconv.Atoi(f.Result["Port"])
	if err != nil {
		h.WriteError("hub/db/config/b: " + err.Error())
		return
	}
	DbDriverAsPfEnum := database.GetSortedDrivers()[number]
	h.Db, err = database.GetdB(DbDriverAsPfEnum, f.Result["Host"], f.Result["Database name"], port,
		f.Result["Username for database access"], f.Result["*Password for database access"])
	h.setDB(DbDriverAsPfEnum, f.Result["Host"], f.Result["Database name"], port,
		f.Result["Username for database access"], f.Result["*Password for database access"])
	if err != nil {
		h.WriteError("hub/db/config/c: " + err.Error())
		return
	}
	h.WriteString(GREEN_OK + "\n")
}

func ServiceDo(serviceToUse *pf.Service, line string) pf.Value {
	v, _ := serviceToUse.Do(line)
	return v
}

var (
	MARGIN         = 92
	GREEN_OK       = ("\033[32mOK\033[0m")
	WAS            = Green("was") + ": "
	GOT            = Red("got") + ": "
	TEST_PASSED    = Green("Test passed!") + "\n"
	VERSION        = "0.6.8"
	BULLET         = "  ▪ "
	BULLET_SPACING = "    " // I.e. whitespace the same width as BULLET.
	GOOD_BULLET    = Green("  ▪ ")
	BROKEN         = Red("  ✖ ")
	PROMPT         = "→ "
	INDENT_PROMPT  = "  "
	ERROR          = text.ERROR
	RT_ERROR       = text.ERROR
	HUB_ERROR      = "<R>Hub error</>: "
)

const HELP = "\nUsage: pipefish [-v | --version] [-h | --help]\n" +
	"                <command> [args]\n\n" +
	"Commands are:\n\n" +
	"  tui           Starts the Pipfish TUI (text user interface).\n" +
	"  run <file>    Runs a Pipefish script if it has a `main` command.\n\n"

func Red(s string) string {
	return "\033[31m" + s + "\033[0m"
}

func Green(s string) string {
	return "\033[32m" + s + "\033[0m"
}

func Cyan(s string) string {
	return "\033[36m" + s + "\033[0m"
}

func Logo() string {
	titleText := " 🧿 Pipefish version " + VERSION + " "
	leftMargin := "  "
	bar := strings.Repeat("═", len(titleText)-2)
	logoString := "\n" +
		leftMargin + "╔" + bar + "╗\n" +
		leftMargin + "║" + titleText + "║\n" +
		leftMargin + "╚" + bar + "╝\n\n"
	return logoString
}

func FlattenedFilename(s string) string {
	base := filepath.Base(s)
	withoutSuffix := strings.TrimSuffix(base, filepath.Ext(base))
	flattened := strings.Replace(withoutSuffix, ".", "_", -1)
	return flattened
}

func (h *Hub) MakeFilepath(scriptFilepath string) string {
	doctoredFilepath := strings.Clone(scriptFilepath)
	if len(scriptFilepath) >= 4 && scriptFilepath[0:4] == "hub/" {
		doctoredFilepath = filepath.Join(settings.PipefishHomeDirectory, filepath.FromSlash(scriptFilepath))
	}
	if len(scriptFilepath) >= 7 && scriptFilepath[0:7] == "rsc-pf/" {
		doctoredFilepath = filepath.Join(settings.PipefishHomeDirectory, "source", "initializer", filepath.FromSlash(scriptFilepath))
	}
	if settings.StandardLibraries.Contains(scriptFilepath) {
		doctoredFilepath = filepath.Join(settings.PipefishHomeDirectory, "source/initializer/libraries/", scriptFilepath)
	}
	if len(scriptFilepath) >= 3 && scriptFilepath[len(scriptFilepath)-3:] != ".pf" && len(scriptFilepath) >= 4 && scriptFilepath[len(scriptFilepath)-4:] != ".hub" {
		doctoredFilepath = doctoredFilepath + ".pf"
	}
	return doctoredFilepath
}
