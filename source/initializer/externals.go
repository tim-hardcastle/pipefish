package initializer

import (
	"github.com/tim-hardcastle/pipefish/source/err"
	"github.com/tim-hardcastle/pipefish/source/settings"
	"github.com/tim-hardcastle/pipefish/source/token"
	"github.com/tim-hardcastle/pipefish/source/values"
	"github.com/tim-hardcastle/pipefish/source/vm"
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
	exValAsString := vm.Do(es.Host, es.Service, line, es.Username, es.Password)
	val := es.Deserializer(exValAsString)
	return val
}

func (es ExternalHttpCallHandler) Problem() *err.Error {
	return nil
}

func (es ExternalHttpCallHandler) GetAPI() string {
	return vm.Do(es.Host, "", "hub serialize \""+es.Service+"\"", es.Username, es.Password)
}

// A function and a couple of types for making an external service call, used to construct 
// an external service.

