package api

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SaviorPhoenix/autobd/helpers"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	ApiVersion string = "v0"
	Version    string = "0.1"
)

var Commit string

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func ServeManifest(w http.ResponseWriter, r *http.Request) {
	helpers.LogHttp(r)
	dir := helpers.GetQueryValue("dir", w, r)
	if dir == "" {
		return
	}
	manifest, err := helpers.GetManifest(dir)
	if err != nil {
		helpers.LogHttpErr(w, r, err, http.StatusInternalServerError)
		return
	}
	serial, _ := json.MarshalIndent(&manifest, "  ", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+Version)
	io.WriteString(w, string(serial))
}

func ServeVersion(w http.ResponseWriter, r *http.Request) {
	type versionInfo struct {
		Ver     string `json:"server"`
		Api     string `json:"api"`
		Commit  string `json:"commit"`
		Comment string `json:"comment"`
	}
	serialVer, _ := json.MarshalIndent(&versionInfo{Version, ApiVersion, Commit,
		"API not intended for human consumption"}, "  ", "  ")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+Version)
	io.WriteString(w, string(serialVer))
}

func ServeSync(w http.ResponseWriter, r *http.Request) {
	helpers.LogHttp(r)
	grab := helpers.GetQueryValue("grab", w, r)
	if grab == "" {
		return
	}
	fd, err := os.Open(grab)
	if err != nil {
		helpers.LogHttpErr(w, r, err, http.StatusInternalServerError)
		return
	}
	defer fd.Close()
	info, err := fd.Stat()
	if err != nil {
		helpers.LogHttpErr(w, r, err, http.StatusInternalServerError)
		return
	}
	if info.IsDir() == true {
		helpers.LogHttpErr(w, r, errors.New("Directory transfer not implemented"),
			http.StatusNotImplemented)
		return
	}
	http.ServeContent(w, r, grab, info.ModTime(), fd)
}

func VersionInfo(commitStr string) {
	if commitStr == "" {
		commitStr = "unknown"
	}
	//Get the commit string from main.go which was populated by the linker
	//this is dumb but I'm too lazy to search for a way to fix it and too
	//stubborn to take it out. having a commit string is nice.
	Commit = commitStr
	fmt.Printf("Autobd version %s (API %s) (git commit %s)\n", Version, ApiVersion, Commit)
}

func SetupRoutes() {
	http.HandleFunc("/"+ApiVersion+"/manifest", GzipHandler(ServeManifest))
	http.HandleFunc("/"+ApiVersion+"/sync", GzipHandler(ServeSync))
	http.HandleFunc("/version", GzipHandler(ServeVersion))
}
