package hub

import (
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
	"net/smtp"
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

	"github.com/tim-hardcastle/pipefish/source/dtypes"
	"github.com/tim-hardcastle/pipefish/source/err"
	"github.com/tim-hardcastle/pipefish/source/initializer"
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
	Sources                map[string][]string
	Db                     *sql.DB
	mailData               mailer
	listeningToHttpOrHttps bool
	// The username and password of the person logged into the terminal.
	TerminalUsername string
	TerminalPassword string
	// The usernames and password of whoever called `hub.Do``.
	username, password string
	// Whether this is an external call.
	store    values.Map
	storekey string
}

type mailer = struct {
	addr   string
	auth   smtp.Auth
	sender string
}

func New(path string, out io.Writer) *Hub {
	h := Hub{
		Services: make(map[string]*pf.Service),
		Out:      out,
	}
	h.OpenHubFile(filepath.Join(path, "hub.hub"))
	return &h
}

func (hub *Hub) CurrentServiceName() string {
	cs := hub.getSV("currentService")
	if cs.T == pf.NULL {
		return ""
	} else {
		return cs.V.(string)
	}
}

func (hub *Hub) hasDatabase() bool {
	return hub.getSV("HUB_DB").V != nil
}

func (hub *Hub) hasMailer() bool {
	return hub.getSV("HUB_MAILER").V != nil
}

func (hub *Hub) getMailer() (string, string, string, string, string, string) {
	mailerStruct := hub.getSV("HUB_MAILER").V.([]pf.Value)
	authStruct := mailerStruct[1].V.([]pf.Value)
	return mailerStruct[0].V.(string), authStruct[0].V.(string), authStruct[1].V.(string), authStruct[2].V.(string), authStruct[3].V.(string), mailerStruct[2].V.(string)
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
func (hub *Hub) getFonts() values.Map {
	theme := hub.getSV("theme")
	if theme.V == nil {
		return values.Map{}
	}
	mapOfThemes := hub.getSV("THEMES").V.(values.Map)
	mapForTheme, themeExists := mapOfThemes.Get(theme)
	if !themeExists {
		return values.Map{}
	}
	fonts := mapForTheme.V.(values.Map)
	return fonts
}

// This takes the input from the REPL, interprets it as a hub command if it begins with 'hub';
// as an instruction to the os if it begins with '$', and as an expression to be passed to
// the current service if none of the above hold.
func (hub *Hub) Do(line, username, password, service string, external bool) {

	// We may be talking to the hub itself.
	hubWords := strings.Fields(line)
	if len(hubWords) > 0 && hubWords[0] == "hub" {
		if len(hubWords) == 1 {
			hub.WriteError("you need to say what you want the hub to do.")
			return
		}
		hub.username = username
		hub.password = password
		hub.setSV("$_external", pf.BOOL, external)
		hub.DoHubCommand(strings.Join(hubWords[1:], " "))
		return
	}

	// We may be talking to the os
	if len(hubWords) > 0 && hubWords[0] == "$" {
		if hub.administered() {
			isAdmin, err := IsUserAdmin(hub.Db, username)
			if err != nil {
				hub.WriteError(err.Error())
				return
			}
			if !isAdmin {
				hub.WriteError("Only administrators can use the shell remotely.")
				return
			}
		} else {
			if external {
				hub.WriteError("on an unadministered hub, for reasons of security and sanity, you can't use the shell remotely.")
				return
			}
		}
		command := exec.Command("sh", "-c", line[2:])
		out, err := command.Output()
		if err != nil {
			hub.WriteError(err.Error())
			return
		}
		if len(out) == 0 {
			hub.WriteString(GREEN_OK)
			return
		}
		hub.WriteString(string(out))
		return
	}
	hub.Sources["REPL input"] = []string{line}
	_, ok := hub.Services[service]
	if !ok {
		hub.WriteError("the hub can't find the service <C>\"" + service + "\"</>.")
		return
	}
	if hub.administered() {
		if !userHasService(hub.Db, username, service) {
			if isAdmin, _ := IsUserAdmin(hub.Db, username); !isAdmin {
				hub.WriteError("you have no access to a service named <C>\"" + service + "\"</> on this hub.")
				return
			}
		}
	}
	hub.ers = []*err.Error{}
	hub.update(service)
	serviceToUse, _ := hub.Services[service]
	// Empty/comment-only lines do nothing, but we wait until now to decide that because we *do* want them to
	// trigger recompilation of code.
	if match, _ := regexp.MatchString(`^\s*(|\/\/.*)$`, line); match {
		hub.WriteString("")
		return
	}
	// The service may be broken, in which case we'll let the empty service handle the input.
	if serviceToUse.IsBroken() {
		serviceToUse = hub.Services[""]
	}

	// We call the service and get the value.
	val := ServiceDo(serviceToUse, line)

	errorsExist, _ := serviceToUse.ErrorsExist()
	if errorsExist { // Any lex-parse-compile errors should end up in the parser of the compiler of the service, returned in p.
		if hub.Services[service].IsBroken() {
			println("\n")
		}
		hub.GetAndReportErrors(serviceToUse)
		return
	}
	hub.outputVal(val, serviceToUse, external)
}

func (hub *Hub) outputVal(val values.Value, serviceToUse *pf.Service, external bool) {
	if val.T == pf.UNSATISFIED_CONDITIONAL {
		hub.WriteError("call returned unsatisfied conditional.")
		return
	}
	if val.T == pf.ERROR && !external {
		e := val.V.(*pf.Error)
		if e.Message == "" {
			e = err.CreateErr(e.ErrorId, e.Token, e.Args...)
		}
		hub.WriteString("\n")
		hub.WritePretty("[" + strconv.Itoa(len(hub.ers)) + "] " + text.ERROR + e.Message + err.DescribePos(e.Token) + ".")
		hub.WriteString("\n\n")
		hub.ers = append(hub.ers, e)
		if len(val.V.(*pf.Error).Values) > 0 {
			hub.WritePretty("Values are available with `hub values`.")
			hub.WriteString("\n\n")
		}
	} else if !serviceToUse.PostHappened() {
		serviceToUse.Output(val)
	}
}

func (hub *Hub) update(serviceName string) {
	if !hub.isLive() {
		return
	}
	path, _ := hub.Services[serviceName].GetFilepath()
	hub.createService(serviceName, path, false)
}

func (hub *Hub) DoHubCommand(line string) {
	hubService := hub.Services["hub"]
	hubReturn := ServiceDo(hubService, line)
	if errorsExist, _ := hubService.ErrorsExist(); errorsExist {
		hub.GetAndReportErrors(hubService)
		return
	}
	hub.outputVal(hubReturn, hubService, false)
}

type hubWriter struct {
	hub *Hub
}

// Things that only make sense if we have RBAM set up.
var rbamVerbs = dtypes.SetOf("add", "change-password", "create-group", "forgot-password", "groups",
	"groups-of-service", "groups-of-user", "let-own", "let-use", "log-off", "log-on",
	"nuke-account", "nuke-admin", "register", "services of group", "services-of-user", "unadd", "uncreate",
	"unlet-own", "unlet-use", "unregister", "users-of-service", "users-of-group")

// Things you can use if you're logged in to a service with RBAM, but not as admin.
var greenList = dtypes.SetOf("change-password", "forgot-password", "log-on", "log-off", "groups",
	"nuke-account", "register", "services", "switch")

func (hw hubWriter) Write(b []byte) (int, error) {
	bits := strings.Split(string(b), ", ")
	verb := bits[0]
	args := bits[1:]
	h := hw.hub
	// There are commands to the hub that should only have permission if you're an administrator, of course.
	// But there are also commands like `switch` which only apply to the person using the TUI, and which won't
	// work if `external` is set.
	username := h.username
	var isAdmin bool
	var err error
	if h.administered() {
		if username == "" && !(verb == "log-on" || verb == "register" || verb == "forgot-password") {
			h.WriteError("this is an administered hub and you aren't logged on. Please use either " +
				"`hub register` to register as a guest; `hub forgot password(username, email string)` " +
				"to replace your password; or `hub sign on` to sign on if you're trying to use the hub on " +
				"the terminal it's running on and you're already registered with this hub.")
			return len(b), nil
		}
		isAdmin, err = IsUserAdmin(h.Db, username)
		if err != nil {
			h.WriteError(err.Error())
			return len(b), nil
		}
		if !isAdmin && !greenList.Contains(verb) {
			h.WriteError("you don't have the admin status necessary to do that.")
			return len(b), nil
		}
	} else {
		if rbamVerbs.Contains(verb) {
			h.WriteError("this hub doesn't have RBAM intitialized.")
			return len(b), nil
		}
	}
	switch verb {
	case "add":
		err := IsUserGroupOwner(h.Db, username, args[1])
		if err != nil {
			h.WriteError(err.Error())
		}
		err = AddUserToGroup(h.Db, args[0], args[1], false)
		if err != nil {
			h.WriteError(err.Error())
		}
	case "api":
		h.update(h.CurrentServiceName())
		h.WriteString(h.Services[h.CurrentServiceName()].Api(h.CurrentServiceName(), h.getFonts(), h.getSV("width").V.(int)))
	case "change-password":
		err = ChangePassword(h.Db, username, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			if h.getSV("$_external").V.(bool) {
				h.WritePretty("You have changed your password. Any connections that relied on the old password are " +
					"now broken, presumably including this one. Please recompile any client services that depended on the old password " +
					"to make them operative again.")
			} else {
				h.TerminalPassword = args[0]
			}
		}
	case "config-admin":
		if h.administered() {
			h.WriteError("this hub is already administered.")
			break
		}
		if !h.hasDatabase() {
			h.WriteError("database has not been configured: edit the `hub.pf` file of this hub to specify a ")
			break
		}
		if !h.hasMailer() && !testing.Testing() {
			h.WriteError("mailer has not been configured: edit the `hub.pf` file of this hub to specify a mailer.")
			break
		}
		if invalid(args) {
			break
		}
		err := AddAdmin(h.Db, args[0], args[1], args[2], args[3], args[4])
		if err != nil {
			h.WriteError(err.Error())
			break
		}
		h.TerminalUsername = args[0]
		h.TerminalPassword = args[4]
		h.WritePretty("You are logged on as <C>" + h.TerminalUsername + "</>.\n")
		h.setSV("isAdministered", pf.BOOL, true)
	case "create-group":
		err := CreateGroup(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		}
		err = AddUserToGroup(h.Db, username, args[0], true)
		if err != nil {
			h.WriteError(err.Error())
		}
	case "dump":
		dump := h.Services[h.CurrentServiceName()].DumpCode(args[0], args[2] == "true")
		h.WriteString("\n" + dump)
		if args[1] == "true" {
			os.WriteFile(filepath.Join(settings.PipefishHomeDirectory, args[3]), []byte(dump), 0666)
		}
	case "env":
		// $_env has been updated by hub.pf. This is called by both `env` and `env delete`.
		env, _ := h.Services["hub"].GetVariable("$_env")
		h.store = env.V.(values.Map)
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
	case "env-wipe":
		h.storekey = ""
		h.store = values.Map{}
		h.SaveAndPropagateHubStore()
	case "errors":
		r, _ := h.Services[h.CurrentServiceName()].GetErrorReport()
		h.WritePretty(r)
	case "forgot-password":
		err := ValidateEmail(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			newPassword := MakePassword()
			ChangePassword(h.Db, args[0], newPassword)
			msg := `Subject: Replacement password for ` + args[0] + "\n" +
				`From: Pipefish mailer (do not reply)

Your replacement password for your account ` + args[0] + ` is ` + newPassword + ".\n\nYou should change this as soon as possible to a new password of your choosing."
			var err error
			if !testing.Testing() {
				err = smtp.SendMail(h.mailData.addr, h.mailData.auth, h.mailData.sender, []string{args[1]}, []byte(msg))
			}
			if err != nil {
				h.WriteError(err.Error())
			} else {
				h.WritePretty("An email with a replacement password has been sent to <C>" + args[1] + "</>.")
			}
		}
	case "groups":
		result, err := GetGroupsOfUser(h.Db, username, true)
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WritePretty(result)
		}
	case "groups-of-service":
		result, err := GetGroupsOfService(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WritePretty(result)
		}
	case "groups-of-user":
		result, err := GetGroupsOfUser(h.Db, args[0], false)
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WritePretty(result)
		}
	case "halt":
		var name string
		_, ok := h.Services[args[0]]
		if ok {
			name = args[0]
		} else {
			h.WriteError("the hub can't find the service <C>\"" + args[0] + "\"</>.")
			break
		}
		if name == "" || name == "hub" {
			h.WriteError("the hub doesn't know what you want to halt.")
			break
		}
		delete(h.Services, name)
		if name == h.CurrentServiceName() {
			h.makeEmptyServiceCurrent()
		}
	case "help":
		h.WriteError("the `hub help` command is temporarily deprecated.")
	case "http":
		h.WriteString(GREEN_OK)
		go h.StartHttp([]string{args[0]}, false)
	case "https":
		if len(args) == 0 {
			h.WriteError("list of domain names cannot be empty.")
		}
		h.WriteString(GREEN_OK)
		go h.StartHttp(args, true)
	case "let-own":
		var inGroup bool
		inGroup, err = IsUserInGroup(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
			break
		}
		if !inGroup {
			err = AddUserToGroup(h.Db, args[0], args[1], true)
			if err != nil {
				h.WriteError(err.Error())
			}
		}
		err = SetOwnership(h.Db, args[0], args[1], true)
		if err != nil {
			h.WriteError(err.Error())
		}
	case "let-use":
		err = LetGroupUseService(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
		}
	case "live-on":
		h.setLive(true)
	case "live-off":
		h.setLive(false)
	case "log-on":
		err := ValidateUser(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
			h.WriteString("Please try again.\n\n")
			break
		}
		h.TerminalUsername = args[0]
		h.TerminalPassword = args[1]
		h.makeEmptyServiceCurrent()
		h.WritePretty("You are logged on as <C>" + h.TerminalUsername + "</>.\n")
	case "log-off":
		h.TerminalUsername = ""
		h.TerminalPassword = ""
		h.makeEmptyServiceCurrent()
		h.WritePretty("<G>OK</>")
		h.WriteString("\n\n" + strings.Repeat("┈", hw.hub.getSV("width").V.(int)) + "\n\n")
		h.WritePretty("This is an administered hub and you aren't logged on. Please use either " +
			"`hub register` to register as a guest; `hub forgot password(username, email string)` " +
			"to replace your password; or `hub sign on` to sign on if you're trying to use the hub on " +
			"the terminal it's running on and you're already registered with this hub.")
		h.WriteString("\n\n")
	case "nuke-account":
		err = UnRegisterUser(h.Db, username)
		if err != nil {
			h.WriteError(err.Error())
		}
		if username == h.TerminalUsername {
			h.TerminalUsername = ""
			h.TerminalPassword = ""
			h.setServiceName("")
		}
	case "nuke-admin":
		DropTables(h.Db)
		h.setSV("isAdministered", pf.BOOL, false)
		h.TerminalUsername = ""
		h.TerminalPassword = ""
		h.setServiceName("")
	case "quit":
		h.Quit()
	case "register":
		if invalid(args) {
			break
		}
		err = AddUser(h.Db, args[0], args[1], args[2], args[3], args[4])
		if err != nil {
			h.WriteError(err.Error())
			break
		}
		err = AddUserToGroup(h.Db, args[0], "Guests", false)
		if err != nil {
			h.WriteError(err.Error())
			break
		}
		h.TerminalUsername = args[0]
		h.TerminalPassword = args[4]
		h.WritePretty("You are logged on as <C>" + h.TerminalUsername + "</>.\n")
	case "reset":
		serviceToReset, ok := h.Services[h.CurrentServiceName()]
		if !ok {
			h.WriteError("the hub can't find the service <C>\"" + h.CurrentServiceName() + "\".")
		}
		if h.CurrentServiceName() == "" {
			h.WriteError("service is empty, nothing to reset.")
		}
		filepath, _ := serviceToReset.GetFilepath()
		h.WritePretty("Restarting script <C>\"" + filepath +
			"\"</> as service <C>\"" + h.CurrentServiceName() + "\"</>.\n")
		h.createService(h.CurrentServiceName(), filepath, true)
	case "run":
		fname := args[0]
		sname := args[1]
		if sname == "" {
			sname = initializer.ExtractFileName(fname)
		}
		if filepath.IsLocal(fname) {
			dir, _ := os.Getwd()
			fname = filepath.Join(dir, fname)
		}
		displayName := filepath.Base(fname)
		if filepath.Ext(displayName) == "" {
			displayName = displayName + ".pf"
		}
		h.WritePretty("Starting script <C>\"" + displayName + "\"</> as service <C>\"" + sname + "\"</>.\n")
		ext := h.getSV("$_external").V.(bool) // Note that we need to do this before createService, which may do external things.
		h.createService(sname, fname, true)
		if !ext {
			h.setServiceName(sname)
			h.tryMain()
		}
	case "serialize":
		h.WriteString(h.Services[args[0]].SerializeApi())
	case "services":
		if h.administered() && !isAdmin {
			result, err := GetServicesOfUser(h.Db, username, true)
			if err != nil {
				h.WriteError(err.Error())
			} else {
				h.WritePretty(result)
			}
		} else {
			if len(h.Services) == 2 {
				h.WriteString("The hub isn't running any services.\n")
			}
			h.WriteString("\n")
			h.list()
		}
	case "services-of-group":
		result, err := GetServicesOfGroup(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WritePretty(result)
		}
	case "services-of-user":
		result, err := GetServicesOfUser(h.Db, args[0], false)
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WritePretty(result)
		}
	case "switch":
		sname := args[0]
		if h.administered() && !isAdmin && !userHasService(h.Db, username, sname) {
			h.WriteError("you have no access to any service named <C>" + sname + "</>.")
			break
		}
		_, ok := h.Services[sname]
		if ok {
			h.setServiceName(sname)
			break
		}
		if !h.administered() || isAdmin {
			h.WriteError("service <C>" + sname + "</> doesn't exist.")
		} else {
			h.WriteError("although you have permissions to use a service called <C>" + sname + "</> on this hub, it's not currently running any service of that name.")
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
	case "log":
		tracking, _ := h.Services[h.CurrentServiceName()].GetTrackingReport()
		h.WritePretty(tracking)
	case "uncreate-group":
		err := UncreateGroup(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		}
	case "unadd-user":
		err := UnAddUserToGroup(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
		}
	case "unlet-use":
		err = UnLetGroupUseService(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
		}
	case "unlet-own":
		var inGroup bool
		inGroup, err = IsUserInGroup(h.Db, args[0], args[1])
		if err != nil {
			h.WriteError(err.Error())
			break
		}
		if !inGroup {
			h.WriteError("user is not in group.")
			break
		}
		err = SetOwnership(h.Db, args[0], args[1], false)
		if err != nil {
			h.WriteError(err.Error())
		}
	case "unregister":
		err = UnRegisterUser(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		}
	case "users-of-group":
		result, err := GetUsersOfGroup(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WritePretty(result)
		}
	case "users-of-service":
		result, err := GetUsersOfService(h.Db, args[0])
		if err != nil {
			h.WriteError(err.Error())
		} else {
			h.WritePretty(result)
		}
	case "values":
		if len(h.ers) == 0 {
			h.WriteError("there are no recent errors.")
			break
		}
		// Usually a runtime error will be the only error, and so necessarily the last one. But also, a runtime error
		// can arise when we're livecoding and we get compilation errors but also a runtime error from whatever we put 
		// into the REPL.
		lastError := h.ers[len(h.ers)-1]
		if lastError.Values == nil {
			h.WriteError("no values were passed.")
			break
		}
		if len(lastError.Values) == 0 {
			h.WriteError("no values were passed.")
			break
		}
		if len(lastError.Values) == 1 {
			h.WriteString("\nThe value passed was:\n\n")
		} else {
			h.WriteString("\nValues passed were:\n\n")
		}
		for _, v := range lastError.Values {
			if v.T == pf.BLING {
				h.WriteString(BULLET_SPACING + h.Services[h.CurrentServiceName()].ToLiteral(v))
			} else {
				h.WriteString(BULLET + h.Services[h.CurrentServiceName()].ToLiteral(v))
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
		h.WritePretty(exp)
		h.WriteString("\n\n")
		refLine := h.GetPretty("Error has reference `\"" + h.ers[num].ErrorId + "\"`.")
		padding := strings.Repeat(" ", h.getSV("width").V.(int)-len(text.StripColors(refLine))-2)
		h.WriteString(padding)
		h.WritePretty(refLine)
		h.WriteString("\n")
	default:
		panic("Unhandled verb " + verb)
	}
	return len(b), nil
}

func invalid(args []string) bool { // TODO --- more validation.
	for _, arg := range args {
		if arg == "" {
			return true
		}
	}
	return false
}

func (hub *Hub) makeWriter() io.Writer {
	return hubWriter{
		hub: hub,
	}
}

func (hub *Hub) Quit() {
	hub.saveHubFile()
	hub.WriteString(GREEN_OK + "\n" + text.Logo() + "Thank you for using Pipefish. Have a nice day!\n\n")
	if !testing.Testing() {
		os.Exit(0)
	}
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

func (hub *Hub) WriteError(s string) {
	hub.WriteString("\n")
	hub.WritePretty(HUB_ERROR + s)
	hub.WriteString("\n\n")
}

func (hub *Hub) WriteString(s string) {
	io.WriteString(hub.Out, s)
	hub.Services["hub"].SetPostHappened()
}

func (hub *Hub) tryMain() { // Guardedly tries to run the `main` command.
	if !hub.Services[hub.CurrentServiceName()].IsBroken() {
		val, _ := hub.Services[hub.CurrentServiceName()].CallMain()
		switch val.T {
		case pf.ERROR:
			hub.WritePretty("\n[0] " + valToString(hub.Services[hub.CurrentServiceName()], val))
			hub.WriteString("\n")
			hub.ers = []*pf.Error{val.V.(*pf.Error)}
		case pf.UNDEFINED_TYPE: // Which is what we get back if there is no `main` command.
		default:
			hub.WriteString(valToString(hub.Services[hub.CurrentServiceName()], val))
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

func (hub *Hub) createService(name, scriptFilepath string, forceUpdate bool) bool {
	needsRebuild := forceUpdate || hub.serviceNeedsUpdate(name)
	if !needsRebuild {
		return false
	}
	newService := pf.NewService()
	newService.SetLocalExternalServices(hub.Services)
	if text.Head(scriptFilepath, "!") {
		scriptFilepath = filepath.Join(settings.PipefishHomeDirectory, scriptFilepath[1:])
	}
	e := newService.InitializeFromFilepathWithStore(scriptFilepath, hub.store) // We get an error only if it completely fails to open the file, otherwise there'll be errors in the Common Parser Bindle as usual.
	hub.Sources, _ = newService.GetSources()
	if newService.IsBroken() {
		if name == "hub" {
			println("Filepath is", scriptFilepath)
			println("Pipefish: unable to compile hub: " + newService.GetErrors()[0].ErrorId + ".")
			println(newService.GetErrors()[0].Message)
			println(err.DescribePos(newService.GetErrors()[0].Token))
			panic("That's all folks!")
		}
		if !newService.IsInitialized() {
			hub.WriteError("unable to open <C>\"" + scriptFilepath + "\"</> with error `" + e.Error() + "`.")
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
	newService.InitializeFromFilepathWithStore(filename, values.Map{})
	if newService.IsBroken() {
		fmt.Println("\nThere were errors running the script " + text.CYAN + "\"" + filename + "\"" + text.RESET + ".\n")
		s, _ := newService.GetErrorReport()
		mdFunc := newService.GetMarkdowner("", 92, values.Map{})
		fmt.Println(mdFunc(s))
		fmt.Println()
		os.Exit(3)
	}
	val, _ := newService.CallMain()
	if val.T == pf.UNDEFINED_TYPE {
		fmt.Println("\nScript <C>\"" + filename + "\"</> has no `main` command.\n\n")
		fmt.Print("\n\nClosing Pipefish.\n\n")
		os.Exit(4)
	}
	if val.T == pf.ERROR {
		fmt.Print(newService.ToString(val))
		os.Exit(86)
	}
	if !newService.PostHappened() {
		fmt.Print(newService.ToString(val)) // Which will be `OK`.
	}
	os.Exit(0)
}

func (hub *Hub) GetAndReportErrors(sv *pf.Service) {
	hub.ers = sv.GetErrors()
	r, _ := sv.GetErrorReport()
	hub.WritePretty(r)
}

func (hub *Hub) CurrentServiceIsBroken() bool {
	return hub.Services[hub.CurrentServiceName()].IsBroken()
}

func (hub *Hub) saveHubFile() string {
	hubService := hub.Services["hub"]
	var buf strings.Builder
	buf.WriteString("var private\n\n")
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
	buf.WriteString("isAdministered = ")
	buf.WriteString(hubService.ToLiteral(hub.getSV("isAdministered")))
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
	h.createService("", "", true)
	h.createService("hub", hubFilepath, true)
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
			storekey := "Default key for testing."
			if !testing.Testing() {
				storekey, _ = rline.Readline()
			}
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
			h.store = h.store.Set(pair.V.([]pf.Value)[0], pair.V.([]pf.Value)[1])
		}
	}
	hubService := h.Services["hub"]
	h.hubFilepath = h.MakeFilepath(hubFilepath)
	v, _ := hubService.GetVariable("allServices")
	services := v.V.(pf.Map).AsSlice()

	if h.hasDatabase() {
		h.Db = h.getSV("HUB_DB").V.(*sql.DB)

	}

	if h.hasMailer() {
		addr, identity, username, password, host, sender := h.getMailer()
		h.mailData = mailer{addr, smtp.PlainAuth(identity, username, password, host), sender}
	}

	errors := false
	for _, pair := range services {
		serviceName := pair.Key.V.(string)
		serviceFilepath := pair.Val.V.(string)
		if serviceName == "" || serviceName == "hub" {
			continue
		}
		h.createService(serviceName, serviceFilepath, true)
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
		srv.SetEnv(hub.store)
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
	Body string
}

func (h *Hub) handleJsonRequest(w http.ResponseWriter, r *http.Request) {
	var request jsonRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if h.administered() && !((!h.listeningToHttpOrHttps) && (request.Body == "hub register" || request.Body == "hub sign on")) {
		err = ValidateUser(h.Db, request.Username, request.Password)
		if err != nil {
			h.WriteError(err.Error())
			return
		}
	}
	var buf bytes.Buffer
	oldOut := h.Out
	h.Out = &buf
	sv := h.Services[request.Service]
	sv.SetOutHandler(sv.MakeLiteralOutHandler(&buf))
	h.Do(request.Body, request.Username, request.Password, request.Service, true)
	h.Out = oldOut
	response := jsonResponse{Body: buf.String()}
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

func (h *Hub) MakeFilepath(scriptFilepath string) string {
	doctoredFilepath := strings.Clone(scriptFilepath)
	if len(scriptFilepath) >= 4 && scriptFilepath[0:4] == "hub/" {
		doctoredFilepath = filepath.Join(settings.PipefishHomeDirectory, filepath.FromSlash(scriptFilepath))
	}
	if len(scriptFilepath) >= 7 && strings.HasPrefix(filepath.ToSlash(scriptFilepath), "rsc-pf/") {
		doctoredFilepath = filepath.Join(settings.PipefishHomeDirectory, "source", "initializer", filepath.FromSlash(scriptFilepath))
	}
	if len(scriptFilepath) >= 3 && scriptFilepath[len(scriptFilepath)-3:] != ".pf" && len(scriptFilepath) >= 4 && scriptFilepath[len(scriptFilepath)-4:] != ".hub" {
		doctoredFilepath = doctoredFilepath + ".pf"
	}
	return doctoredFilepath
}

func (h *Hub) administered() bool {
	return h.getSV("isAdministered").V.(bool)
}
