package utils

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/options"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
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
		_, filepath, line, _ := runtime.Caller(1)
		_, file := path.Split(filepath)
		log.Errorf("HttpErrorHandler()->[file:%s line:%d]: %s", file, line, err.Error())
		serialErr, _ := json.Marshal(err.Error())
		http.Error(h.Response, string(serialErr), httpStatus)
	}
	return (err != nil)
}

// HandlePanic _Never_ returns on error, instead it panics
func HandlePanic(err error) {
	if err != nil {
		_, filepath, line, _ := runtime.Caller(1)
		_, file := path.Split(filepath)
		log.Panicf("[file:%s line:%d]: %s", file, line, err.Error())
	}
}

func HandleError(err error, action int) bool {
	if err != nil {
		_, filepath, line, _ := runtime.Caller(1)
		_, file := path.Split(filepath)
		switch action {
		case ErrorActionErr:
			log.Errorf("[file:%s line:%d]: %s", file, line, err.Error())
			break
		case ErrorActionWarn:
			log.Warnf("[file:%s line:%d]: %s", file, line, err.Error())
			break
		case ErrorActionDebug:
			log.Debugf("[file:%s line:%d]: %s", file, line, err.Error())
			break
		case ErrorActionInfo:
			log.Infof("[file:%s line:%d]: %s", file, line, err.Error())
			break
		}
	}
	return (err != nil)
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
