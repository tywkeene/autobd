package utils

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
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
