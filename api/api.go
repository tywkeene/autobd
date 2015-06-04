//Package api implements the endpoints and utilities necessary to present
//a consistent API to autobd-nodes
//Currently there are three endpoints:
//
//NOTE: All endpoints return a JSON encoded error string on error via logging.LogHttpErr()
//
// /<version>/manifest?dir=<dirname>
//
// /<version>/sync?grab=<file or directory name>
//
// /version
//
//The '/manifest' endpoint takes a directory as an argument passed as the url
//paramater 'dir'. It returns a JSON encoded map of strings->manifest.Manifest
//of the requested directory or a JSON encoded error string on error
//
//The '/sync' endpoint takes a directory or filename as an argument pass as the url
//parameter 'grab'. If 'grab' is a directory, it will set the HTTP header 'Content-Type-'
//to 'application/x-tar', and transport the directory packed as a tarball to the client via packing.PackDir()
//
//The '/version' endpoint simple returns a JSON version.VersionInfo struct
//
package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/tywkeene/autobd/logging"
	"github.com/tywkeene/autobd/manifest"
	"github.com/tywkeene/autobd/packing"
	"github.com/tywkeene/autobd/version"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

//Check and make sure the client wants or can handle gzip, and replace the writer if it
//can, if not, simply use the normal http.ResponseWriter
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

//GetQueryValue() takes a name of a key:value pair to fetchf rom a URL encoded query,
//a http.ResponseWriter 'w', and a http.Request 'r'. In the event that an error is encountered
//the error will be returned to the client via logging facilities that use 'w' and 'r'
func GetQueryValue(name string, w http.ResponseWriter, r *http.Request) string {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Println(err)
		logging.LogHttpErr(w, r, fmt.Errorf("Query parse error"), http.StatusInternalServerError)
		return ""
	}
	value := query.Get(name)
	if len(value) == 0 || value == "" {
		log.Println(err)
		logging.LogHttpErr(w, r, fmt.Errorf("Must specify %s", name), http.StatusBadRequest)
		return ""
	}
	return value
}

//ServeManifest() is the http handler for the "/manifest" API endpoint.
//It takes the requested directory passed as a url parameter "dir" i.e "/manifest?dir=/"
//
//It will then generate a manifest by calling api.GetQueryValue(), then writes it to the client as a
//map[string]*manifest.Manifest encoded in json
func ServeManifest(w http.ResponseWriter, r *http.Request) {
	logging.LogHttp(r)
	dir := GetQueryValue("dir", w, r)
	if dir == "" {
		return
	}
	dirManifest, err := manifest.GetManifest(dir)
	if err != nil {
		log.Println(err)
		logging.LogHttpErr(w, r, fmt.Errorf("Error getting manifest"), http.StatusInternalServerError)
		return
	}
	serial, _ := json.MarshalIndent(&dirManifest, "  ", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+version.Server())
	io.WriteString(w, string(serial))
}

//ServeServerVer() is the http handler for the "/version" http API endpoint.
//It writes the json encoded struct version.VersionInfo to the client
func ServeServerVer(w http.ResponseWriter, r *http.Request) {
	logging.LogHttp(r)
	serialVer, _ := json.MarshalIndent(&version.VersionInfo{version.Server(), version.Commit()}, "  ", "  ")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+version.Server())
	io.WriteString(w, string(serialVer))
}

//ServeSync() is the http handler for the "/sync" http API endpoint.
//It takes the requested directory or file name passed as a url parameter "grab" i.e "/sync?grab=file1"
//
//If the requested file is a directory, it will be tarballed and the "Content-Type" http-header will be
//set to "application/x-tar".
//If the file is a normal file, it will be served with http.ServeContent(), with the Content-Type http-header
//set by http.ServeContent()
func ServeSync(w http.ResponseWriter, r *http.Request) {
	logging.LogHttp(r)
	grab := GetQueryValue("grab", w, r)
	if grab == "" {
		return
	}
	fd, err := os.Open(grab)
	if err != nil {
		log.Println(err)
		logging.LogHttpErr(w, r, fmt.Errorf("Error getting file"), http.StatusInternalServerError)
		return
	}
	defer fd.Close()
	info, err := fd.Stat()
	if err != nil {
		log.Println(err)
		logging.LogHttpErr(w, r, fmt.Errorf("Error getting file"), http.StatusInternalServerError)
		return
	}
	if info.IsDir() == true {
		w.Header().Set("Content-Type", "application/x-tar")
		if err := packing.PackDir(grab, w); err != nil {
			log.Println(err)
			logging.LogHttpErr(w, r, fmt.Errorf("Error packing directory"), http.StatusInternalServerError)
			return
		}
		return
	}
	http.ServeContent(w, r, grab, info.ModTime(), fd)
}

func SetupRoutes() {
	http.HandleFunc("/"+"v"+version.Major()+"/manifest", GzipHandler(ServeManifest))
	http.HandleFunc("/"+"v"+version.Major()+"/sync", GzipHandler(ServeSync))
	http.HandleFunc("/version", GzipHandler(ServeServerVer))
}
