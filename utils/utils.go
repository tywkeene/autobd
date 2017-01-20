package utils

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/options"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type HttpErrorHandler struct {
	Caller   string
	Response http.ResponseWriter
	Request  *http.Request
}

const (
	ErrorActionErr = iota
	ErrorActionWarn
	ErrorActionDebug
	ErrorActionInfo
)

func NewHttpErrorHandle(caller string, response http.ResponseWriter, request *http.Request) *HttpErrorHandler {
	return &HttpErrorHandler{caller, response, request}
}

// h.Handle checks the err in h *HttpHandleError, if there is an error, the error is logged in
// HandleError locally, according to the action passed to h.Handle, and then serialized
// in json and sent to the remote address via http, then returns true.
// Otherwise, if there is no error, h.Handle returns false
func (h *HttpErrorHandler) Handle(err error, httpStatus int, action int) bool {
	if err != nil {
		HandleError("utils/h.Handle()->"+h.Caller, fmt.Errorf("%s: %s", h.Request.RemoteAddr, err.Error()), action)
		serialErr, _ := json.Marshal(err.Error())
		http.Error(h.Response, string(serialErr), httpStatus)
	}
	return (err != nil)
}

// HandlePanic _Never_ returns on error, instead it panics
func HandlePanic(caller string, err error) {
	if err != nil {
		log.Panicf("%s: %s", caller, err)
	}
}

func HandleError(caller string, err error, action int) bool {
	if err != nil {
		switch action {
		case ErrorActionErr:
			log.Errorf("%s: %s", caller, err.Error())
			break
		case ErrorActionWarn:
			log.Warnf("%s: %s", caller, err.Error())
			break
		case ErrorActionDebug:
			log.Debugf("%s: %s", caller, err.Error())
			break
		case ErrorActionInfo:
			log.Infof("%s: %s", caller, err.Error())
			break
		}
	}
	return (err != nil)
}

func WriteJson(path string, data interface{}) error {
	outfile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outfile.Close()
	serial, err := json.MarshalIndent(&data, " ", " ")
	if err != nil {
		return err
	}
	_, err = outfile.WriteString(string(serial))
	return err
}

func ReadJson(path string, data interface{}) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	serial, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(serial, &data); err != nil {
		return err
	}
	return nil
}

func WriteFile(filename string, source io.Reader) error {
	writer, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writer.Close()
	io.Copy(writer, source)
	return nil
}

// This is neat: https://coderwall.com/p/cp5fya/measuring-execution-time-in-go
func TimeTrack(start time.Time, name string) {
	if options.Config.LogTimeTrack == true {
		elapsed := time.Since(start)
		log.Infof("%s took %s", name, elapsed)
	}
}
