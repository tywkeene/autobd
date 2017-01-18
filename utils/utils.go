package utils

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/options"
	"io"
	"io/ioutil"
	"os"
	"time"
)

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
