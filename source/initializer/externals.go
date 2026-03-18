package initializer

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/tim-hardcastle/pipefish/source/err"
	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/token"
	"github.com/tim-hardcastle/pipefish/source/values"
)

// We have two types of external service, defined below: one for services on the same hub, one for services on
// a different hub.

type ExternalCallToHubHandler struct {
	Evaluator    func(line string) values.Value
	ProblemFn    func() bool
	SerializeApi func() string
}

func (ex ExternalCallToHubHandler) Evaluate(line string) values.Value {
	if settings.SHOW_XCALLS {
		println("Line is", line)
	}
	return ex.Evaluator(line)
}

func (es ExternalCallToHubHandler) Problem() *err.Error {
	if es.ProblemFn() {
		return err.CreateErr("ext/broken", &token.Token{Source: "Pipefish builder"})
	}
	return nil
}

func (es ExternalCallToHubHandler) GetAPI() string {
	return es.SerializeApi()
}

type ExternalHttpCallHandler struct {
	Host         string
	Service      string
	Username     string
	Password     string
	Deserializer func(valAsString string) values.Value
}

func (es ExternalHttpCallHandler) Evaluate(line string) values.Value {
	if settings.SHOW_XCALLS {
		println("Line is", line)
	}
	exValAsString := Do(es.Host, es.Service, line, es.Username, es.Password)
	val := es.Deserializer(exValAsString)
	return val
}

func (es ExternalHttpCallHandler) Problem() *err.Error {
	return nil
}

func (es ExternalHttpCallHandler) GetAPI() string {
	return Do(es.Host, "", "hub serialize \""+es.Service+"\"", es.Username, es.Password)
}

// A function and a couple of types for making an external service call, used to construct 
// an external service.

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

func Do(host, service, line, username, password string) string {
	jRq := jsonRequest{Body: line, Service: service, Username: username, Password: password}
	body, _ := json.Marshal(jRq)
	request, err := http.NewRequest("POST", host, bytes.NewBuffer(body))
	if err != nil {
		return "error \"Can't parse request\"" // Obviously this one shouldn't happen.
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "error: `" + err.Error() + "`"
	}

	defer response.Body.Close()
	rBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "error: " + err.Error()
	}
	if settings.SHOW_XCALLS {
		rawJ := ""
		for _, c := range rBody {
			rawJ = rawJ + (string(c))
		}
		println("Raw json is", rawJ)
	}
	var jRsp jsonResponse
	err = json.Unmarshal(rBody, &jRsp)
	if err != nil {
		return "error: " + err.Error()
	}
	return jRsp.Body
}