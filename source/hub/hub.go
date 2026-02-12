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
	return hub.getSV("$_db").V != nil
}

func (hub *Hub) hasMailer() bool {
	return hub.getSV("$_mailer").V != nil
}

// Temporary thing until all SQL io is in hub.pf.
func (hub *Hub) getDB() (string, string, int, string, string, string) {
	dbStruct := hub.getSV("$_db").V.([]pf.Value)
	driver := hub.Services["hub"].ToLiteral(dbStruct[0])
	return driver, dbStruct[1].V.(string), dbStruct[2].V.(int), dbStruct[3].V.(string), dbStruct[4].V.(string), dbStruct[5].V.(string)
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
	hub.outputVal(val, serviceToUse, external)
	return passedServiceName, false
}

func (hub *Hub) outputVal(val values.Value, serviceToUse *pf.Service, external bool) {
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
}

func (hub *Hub) update() {
	needsUpdate := hub.serviceNeedsUpdate(hub.currentServiceName())
	if hub.isLive() && needsUpdate {
		path, _ := hub.Services[hub.currentServiceName()].GetFilepath()
		hub.StartAndMakeCurrent(hub.TerminalUsername, hub.currentServiceName(), path)
	}
}

func (hub *Hub) DoHubCommand(line string) { // TODO --- this is where we need to pass in whether it's external.
	hubService := hub.Services["hub"]
	hubReturn := ServiceDo(hubService, line)
	if errorsExist, _ := hubService.ErrorsExist(); errorsExist { 
		hub.GetAndReportErrors(hubService)
		return
	}
	hub.outputVal(hubReturn, hubService, false)
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
	h := hw.hub
	username := h.username
	var isAdmin bool
	var err error
	if h.isAdministered() {
		isAdmin, err = database.IsUserAdmin(h.Db, username)
		if err != nil {
			h.WriteError(err.Error())
			return len(b), nil
		}
		if !isAdmin && !greenList.Contains(verb) {
			h.WriteError("you don't have the admin status necessary to do that.")
			return len(b), nil
		}
		if username == "" && !(verb == "log-on" || verb == "register" || verb == "quit") {
			h.WriteError("\nthis is an administered hub and you aren't logged on. Please enter either " +
				"`hub register` to register as a guest, or `hub log on` to log on if you're already registered " +
				"with this hub.")
			return len(b), nil
		}
	} else {
		if rbamVerbs.Contains(verb) {
			h.WriteError("this hub doesn't have RBAM intitialized.")
		}
	}
	switch verb {
	case "add":
		err := database.IsUserGroupOwner(h.Db, username, args[1])
		if err != nil {
			h.WriteError(err.Error())
		}
		err = database.AddUserToGroup(h.Db, args[0], args[1], false)
		if err != nil {
			h.WriteError(err.Error())
		}
	case "api":
		h.update()
		h.WriteString(h.Services[h.currentServiceName()].Api(h.currentServiceName(), h.getFonts(), h.getSV("width").V.(int)))
	case "config-admin":
		if h.isAdministered() {
			h.WriteError("this hub is already administered.")
			break
		}
		if h.Db == nil {
		h.WriteError("database has not been configured: edit the `hub.usr` file of this hub to specify a database and a mailer.")
		break
	}
	err := database.AddAdmin(h.Db, args[0], args[1], args[2], args[3], args[4], h.currentServiceName(), settings.PipefishHomeDirectory)
	if err != nil {
		h.WriteError(err.Error())
		break
	}
	h.TerminalUsername = args[0]
	h.TerminalPassword = args[4]
	h.WritePretty("You are logged on as " + h.TerminalUsername + ".\n")
	h.administered = true
	case "create":
		err := database.AddGroup(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		}
		err = database.AddUserToGroup(h.Db, username, args[0], true)
		if err != nil {
			h.WriteError(err.Error())
		}
		
	case "edit":
		command := exec.Command("vim", args[0])
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		err := command.Run()
		if err != nil {
			h.WriteError(err.Error())
		}
	case "env":
		// $_env has been updated by hub.pf. This is called by both `env` and `env delete`.
		env, _ := h.Services["hub"].GetVariable("$_env")
		h.store = *env.V.(*values.Map)
		h.SaveAndPropagateHubStore()
		
	case "env-key":
		cur := args[0]
		new := args[1]	
		if cur != h.storekey {
			h.WriteError("incorrect environment key.")
			break
		}
		h.storekey = new
		h.SaveAndPropagateHubStore()
		h.WriteString(GREEN_OK + "\n")
	case "env-wipe":
		h.storekey = ""
		h.store = values.Map{}
		h.SaveAndPropagateHubStore()
		
	case "errors":
		r, _ := h.Services[h.currentServiceName()].GetErrorReport()
		h.WritePretty(r)
	case "groups-of-user":
		result, err := database.GetGroupsOfUser(h.Db, args[0], false)
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WriteString(result)
		}
	case "groups-of-service":
		result, err := database.GetGroupsOfService(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WriteString(result)
		}
	case "halt":
		var name string
		_, ok := h.Services[args[0]]
		if ok {
			name = args[0]
		} else {
			h.WriteError("the hub can't find the service <C>\"" + args[0] + "\"</>.")
		}
		if name == "" || name == "hub" {
			h.WriteError("the hub doesn't know what you want to stop.")
		}
		delete(h.Services, name)
		
		if name == h.currentServiceName() {
			h.makeEmptyServiceCurrent()
		}
	case "help":
		if helpMessage, ok := helpStrings[args[0]]; ok {
			h.WritePretty(helpMessage + "\n")
		} else {
			h.WriteError("the `hub help` command doesn't accept " +
				"`\"" + args[0] + "\"` as a parameter.")
		}
	case "http":
		h.WriteString(GREEN_OK)
		go h.StartHttp([]string{args[0]}, false)
	case "https":
		if len(args) == 0 {
			h.WriteError("list of domain names cannot be empty")
		}
		h.WriteString(GREEN_OK)
		domains := []string{}
		for _, arg := range args {
			domains = append(domains, arg)
		}
		go h.StartHttp(domains, true)
	case "let":
		err = database.LetGroupUseService(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
		}
		
	case "live-on":
		h.setLive(true)
	case "live-off":
		h.setLive(false)
	case "log":
		tracking, _ := h.Services[h.currentServiceName()].GetTrackingReport()
		h.WritePretty(tracking)
		h.WriteString("\n")
	case "log-on":
		_, err := database.ValidateUser(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
			h.WriteString("Please try again.\n\n")
			break
		}
		h.TerminalUsername = args[0]
		h.TerminalPassword = args[1]
		h.WriteString(GREEN_OK + "\n")
	case "log-off":
		h.TerminalUsername = ""
		h.TerminalPassword = ""
		h.makeEmptyServiceCurrent()
		h.WritePretty("\nThis is an administered hub and you aren't logged on. Please enter either " +
			"`hub register` to register as a user, or `hub log on` to log on if you're already registered " +
			"with this hub.\n\n")
	case "groups":
		result, err := database.GetGroupsOfUser(h.Db, username, true)
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WriteString(result)
		}
	case "quit":
		h.Quit()
	case "register":
		err = database.AddUser(h.Db, args[0], args[1], args[2], args[3], args[4], "")
		if err != nil {
			h.WriteError(err.Error())
			break
		}
		err = database.AddUserToGroup(h.Db, args[0], "Guests", false)
		if err != nil {
			h.WriteError(err.Error())
			break
		}
		h.TerminalUsername = args[0]
		h.TerminalPassword = args[4]
		h.WriteString("You are logged in as " + h.TerminalUsername + ".\n")
	case "replay":
		h.oldServiceName = h.currentServiceName()
		h.playTest(args[0], false)
		h.setServiceName(h.oldServiceName)
		_, ok := h.Services["#test"]
		if ok {
			delete(h.Services, "#test")
		}
	case "replay-diff":
		h.oldServiceName = h.currentServiceName()
		h.playTest(args[0], true)
		h.setServiceName(h.oldServiceName)
		_, ok := h.Services["#test"]
		if ok {
			delete(h.Services, "#test")
		}
	case "reset":
		serviceToReset, ok := h.Services[h.currentServiceName()]
		if !ok {
			h.WriteError("the hub can't find the service <C>\"" + h.currentServiceName() + "\".")
		}
		if h.currentServiceName() == "" {
			h.WriteError("service is empty, nothing to reset.")
		}
		filepath, _ := serviceToReset.GetFilepath()
		h.WritePretty("Restarting script <C>\"" + filepath +
			"\"</> as service <C>\"" + h.currentServiceName() + "\"</>.\n")
		h.StartAndMakeCurrent(username, h.currentServiceName(), filepath)
		h.lastRun = []string{h.currentServiceName()}
	case "rerun":
		if len(h.lastRun) == 0 {
			h.WriteError("nothing to rerun.")
		}
		filepath, _ := h.Services[h.lastRun[0]].GetFilepath()
		h.WritePretty("Rerunning script <C>\"" + filepath +
			"</>\" as service <C>\"" + h.lastRun[0] + "\"</>.\n")
		h.StartAndMakeCurrent(username, h.lastRun[0], filepath)
		h.tryMain()
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
		h.lastRun = []string{fname, sname}
		h.WritePretty("Starting script <C>\"" + filepath.Base(fname) + "\"</> as service <C>\"" + sname + "\"</>.\n")
		h.StartAndMakeCurrent(username, sname, fname)
		h.tryMain()
	case "serialize":
		h.WriteString(h.Services[args[0]].SerializeApi())
	case "services":
		if h.isAdministered() {
			result, err := database.GetServicesOfUser(h.Db, username, true)
			if err != nil {
				h.WriteError(err.Error())
			} else {
				h.WriteString(result)
			}
		} else {
			if len(h.Services) == 2 {
				h.WriteString("The hub isn't running any services.\n")
			}
			h.WriteString("\n")
			h.list()
		}
	case "services-of-user":
		result, err := database.GetServicesOfUser(h.Db, args[0], false)
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WriteString(result)
		}
	case "services-of-group":
		result, err := database.GetServicesOfGroup(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WriteString(result)
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
		h.snap = NewSnap(scriptFilepath, testFilepath)
		h.oldServiceName = h.currentServiceName()
		if h.StartAndMakeCurrent(username, "#snap", scriptFilepath) {
			snapService := h.Services["#snap"]
			ty, _ := snapService.TypeNameToType("$_OutputAs")
			snapService.SetVariable("$_outputAs", ty, 0)
			h.WriteString("Serialization is ON.\n")
			in, out := MakeSnapIo(snapService, h.Out, h.snap)
			currentService := snapService
			currentService.SetInHandler(in)
			currentService.SetOutHandler(out)
		} else {
			h.WriteError("failed to start snap")
		}
	case "snap-good":
		if h.currentServiceName() != "#snap" {
			h.WriteError("you aren't taking a snap.")
		}
		result := h.snap.Save(GOOD)
		h.WriteString(result + "\n")
		h.setServiceName(h.oldServiceName)
	case "snap-bad":
		if h.currentServiceName() != "#snap" {
			h.WriteError("you aren't taking a snap.")
		}
		result := h.snap.Save(BAD)
		h.WriteString(result + "\n")
		h.setServiceName(h.oldServiceName)
	case "snap-record":
		if h.currentServiceName() != "#snap" {
			h.WriteError("you aren't taking a snap.")
		}
		result := h.snap.Save(RECORD)
		h.WriteString(result + "\n")
		h.setServiceName(h.oldServiceName)
	case "snap-discard":
		if h.currentServiceName() != "#snap" {
			h.WriteError("you aren't taking a snap.")
		}
		
		h.setServiceName(h.oldServiceName)
	case "switch":
		sname := args[0]
		_, ok := h.Services[sname]
		if ok {
			
			if h.administered {
				access, err := database.DoesUserHaveAccess(h.Db, username, sname)
				if err != nil {
					h.WriteError("o/ " + err.Error())
				}
				if !access {
					h.WriteError("you don't have access to service <C>\"" + sname + "\"</>.")
				}
				database.UpdateService(h.Db, username, sname)
			} else {
				h.setServiceName(sname)
			}
			break
		}
		h.WriteError("service <C>\"" + sname + "\"</> doesn't exist.")
	case "test":
		fname := args[0]
		if filepath.IsLocal(fname) {
			dir, _ := os.Getwd()
			fname = filepath.Join(dir, fname)
		}
		file, err := os.Open(fname)
		if err != nil {
			h.WriteError(strings.TrimSpace(err.Error()) + "\n")
			break
		}
		defer file.Close()
		fileInfo, err := file.Stat()
		if err != nil {
			h.WriteError(strings.TrimSpace(err.Error()) + "\n")
			break
		}
		if fileInfo.IsDir() {
			files, err := file.Readdir(0)
			if err != nil {
				h.WriteError(strings.TrimSpace(err.Error()) + "\n")
				break
			}
			for _, potentialPfFile := range files {
				if filepath.Ext(potentialPfFile.Name()) == ".pf" {
					h.TestScript(fname+"/"+potentialPfFile.Name(), ERROR_CHECK)
				}
			}
		} else {
			h.TestScript(fname, ERROR_CHECK)
		}
	case "trace":
		if len(h.ers) == 0 {
			h.WriteError("there are no recent errors.")
			break
		}
		if len(h.ers[0].Trace) == 0 {
			h.WriteError("not a runtime error.")
			break
		}
		h.WritePretty(pf.GetTraceReport(h.ers[0]))
	case "users-of-group":
		result, err := database.GetUsersOfGroup(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WriteString(result)
		}
	case "users-of-service":
		result, err := database.GetUsersOfService(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WriteString(result)
		}
	case "values":
		if len(h.ers) == 0 {
			h.WriteError("there are no recent errors.")
			break
		}
		if h.ers[0].Values == nil {
			h.WriteError("no values were passed.")
			break
		}
		if len(h.ers[0].Values) == 0 {
			h.WriteError("no values were passed.")
			break
		}
		if len(h.ers[0].Values) == 1 {
			h.WriteString("\nThe value passed was:\n\n")
		} else {
			h.WriteString("\nValues passed were:\n\n")
		}
		for _, v := range h.ers[0].Values {
			if v.T == pf.BLING {
				h.WriteString(BULLET_SPACING + h.Services[h.currentServiceName()].ToLiteral(v))
			} else {
				h.WriteString(BULLET + h.Services[h.currentServiceName()].ToLiteral(v))
			}
			h.WriteString("\n")
		}
		h.WriteString("\n")
	case "where":
		num, _ := strconv.Atoi(args[0])
		if num < 0 {
			h.WriteError("the `where` keyword can't take a negative number as a parameter.")
			break
		}
		if num >= len(h.ers) {
			h.WriteError("there aren't that many errors.")
			break
		}
		println()
		if h.ers[num].Token.Line <= 0 {
			h.WriteError("line number is not available.")
		}
		line := h.Sources[h.ers[num].Token.Source][h.ers[num].Token.Line-1] + "\n"
		startUnderline := h.ers[num].Token.ChStart
		lenUnderline := h.ers[num].Token.ChEnd - startUnderline
		if lenUnderline == 0 {
			lenUnderline = 1
		}
		endUnderline := startUnderline + lenUnderline
		h.WriteString(line[0:startUnderline])
		h.WriteString(Red(line[startUnderline:endUnderline]))
		h.WriteString(line[endUnderline:])
		h.WriteString(strings.Repeat(" ", startUnderline))
		h.WriteString(Red(strings.Repeat("▔", lenUnderline)))
	case "why":
		h.WriteString("\n")
		num, _ := strconv.Atoi(args[0])
		if num >= len(h.ers) {
			h.WriteError("there aren't that many errors.")
			break
		}
		exp, _ := pf.ExplainError(h.ers, num)
		h.WritePretty("<R>Error</>: " + h.ers[num].Message + ".")
		h.WriteString("\n\n")
		h.WritePretty(exp + "\n\n")
		refLine := h.GetPretty("Error has reference `\"" + h.ers[num].ErrorId + "\"`.")
		padding := strings.Repeat(" ", h.getSV("width").V.(int)-len(text.StripColors(refLine))-2)
		h.WriteString(padding)
		h.WritePretty(refLine)
		h.WriteString("\n")
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
	hub.Services["hub"].SetPostHappened() 
}

var helpStrings = map[string]string{}

var helpTopics = []string{}

func init() {
	helpStrings = map[string]string{}
}

func (hub *Hub) StartAndMakeCurrent(username, serviceName, scriptFilepath string) bool {
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
			println(text.DescribePos(newService.GetErrors()[0].Token))
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

func (hub *Hub) saveHubFile() string {
	hubService := hub.Services["hub"]
	var buf strings.Builder
	buf.WriteString("var\n\n")
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

	if h.hasDatabase() {
		driver, hostpath, port, hostname, username, password := h.getDB()
		h.Db, _ = database.GetdB(driver, hostpath, port, hostname, username, password)
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
