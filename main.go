package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

type File struct {
	Name     string           `json:"name"`
	Size     int64            `json:"size"`
	ModTime  time.Time        `json:"lastModified"`
	Mode     os.FileMode      `json:"fileMode"`
	IsDir    bool             `json:"isDir"`
	Manifest map[string]*File `json:"manifest,omitempty"`
}

func NewFile(name string, size int64, modtime time.Time, mode os.FileMode, isDir bool) *File {
	return &File{name, size, modtime, mode, isDir, nil}
}

func GetManifest(dirPath string) (map[string]*File, error) {
	list, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	manifest := make(map[string]*File)
	for _, child := range list {
		childPath := path.Join(dirPath, child.Name())
		manifest[childPath] = NewFile(childPath, child.Size(), child.ModTime(), child.Mode(), child.IsDir())
		if child.IsDir() == true {
			childContent, err := GetManifest(childPath)
			if err != nil {
				return nil, err
			}
			manifest[childPath].Manifest = childContent
		}
	}
	return manifest, nil
}

func LogHttp(r *http.Request) {
	log.Printf("%s %s %s %s", r.Method, r.URL, r.RemoteAddr, r.UserAgent())
}

func LogHttpErr(w http.ResponseWriter, r *http.Request, err error, status int) {
	log.Printf("Returned error \"%s\" (HTTP %s) to %s", err.Error(), http.StatusText(status), r.RemoteAddr)
	http.Error(w, err.Error(), status)
}

func GetQueryValue(name string, w http.ResponseWriter, r *http.Request) string {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		LogHttpErr(w, r, err, http.StatusInternalServerError)
		return ""
	}

	value := query.Get(name)
	if len(value) == 0 || value == "" {
		LogHttpErr(w, r, fmt.Errorf("Must specify %s", name), http.StatusBadRequest)
		return ""
	}
	return value
}

func ServeManifest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	LogHttp(r)

	dir := GetQueryValue("dir", w, r)
	if dir == "" {
		LogHttpErr(w, r, errors.New("Must specify directory"), http.StatusBadRequest)
		return
	}
	manifest, err := GetManifest(dir)
	if err != nil {
		LogHttpErr(w, r, err, http.StatusInternalServerError)
		return
	}
	serial, _ := json.MarshalIndent(&manifest, "  ", "  ")

	elapsed := time.Since(start)
	log.Printf("-->\tServeManifest(%s) enumeration and marshaling took %s", dir, elapsed)

	writeStart := time.Now()
	io.WriteString(w, string(serial))
	writeElapsed := time.Since(writeStart)
	log.Printf("-->\tServeManifest(%s) write json took %s", dir, writeElapsed)
}

func main() {
	http.HandleFunc("/v0/manifest", ServeManifest)
	log.Panic(http.ListenAndServe(":8080", nil))
}
