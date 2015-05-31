package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/tywkeene/autobd/helpers"
	"github.com/tywkeene/autobd/version"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

type ServerVerInfo struct {
	Ver     string `json:"server"`
	Api     string `json:"api"`
	Commit  string `json:"commit"`
	Comment string `json:"comment"`
}

type Manifest struct {
	Name    string               `json:"name"`
	Size    int64                `json:"size"`
	ModTime time.Time            `json:"lastModified"`
	Mode    os.FileMode          `json:"fileMode"`
	IsDir   bool                 `json:"isDir"`
	Files   map[string]*Manifest `json:"files,omitempty"`
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

func NewManifest(name string, size int64, modtime time.Time, mode os.FileMode, isDir bool) *Manifest {
	return &Manifest{name, size, modtime, mode, isDir, nil}
}

func GetManifest(dirPath string) (map[string]*Manifest, error) {
	list, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	manifest := make(map[string]*Manifest)
	for _, child := range list {
		childPath := path.Join(dirPath, child.Name())
		manifest[childPath] = NewManifest(childPath, child.Size(), child.ModTime(), child.Mode(), child.IsDir())
		if child.IsDir() == true {
			childContent, err := GetManifest(childPath)
			if err != nil {
				return nil, err
			}
			manifest[childPath].Files = childContent
		}
	}
	return manifest, nil
}

func ServeManifest(w http.ResponseWriter, r *http.Request) {
	helpers.LogHttp(r)
	dir := helpers.GetQueryValue("dir", w, r)
	if dir == "" {
		return
	}
	manifest, err := GetManifest(dir)
	if err != nil {
		helpers.LogHttpErr(w, r, fmt.Errorf("Error getting manifest"), http.StatusInternalServerError)
		return
	}
	serial, _ := json.MarshalIndent(&manifest, "  ", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+version.Server())
	io.WriteString(w, string(serial))
}

func ServeServerVer(w http.ResponseWriter, r *http.Request) {
	helpers.LogHttp(r)
	serialVer, _ := json.MarshalIndent(&ServerVerInfo{version.Server(), version.API(), version.Commit(),
		"API not intended for human consumption"}, "  ", "  ")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+version.Server())
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
		helpers.LogHttpErr(w, r, fmt.Errorf("Error getting file"), http.StatusInternalServerError)
		return
	}
	defer fd.Close()
	info, err := fd.Stat()
	if err != nil {
		helpers.LogHttpErr(w, r, fmt.Errorf("Error getting file"), http.StatusInternalServerError)
		return
	}
	if info.IsDir() == true {
		w.Header().Set("Content-Type", "application/x-tar")
		if err := helpers.PackDir(grab, w); err != nil {
			helpers.LogHttpErr(w, r, fmt.Errorf("Error packing directory"), http.StatusInternalServerError)
			return
		}
		return
	}
	http.ServeContent(w, r, grab, info.ModTime(), fd)
}

func SetupRoutes() {
	http.HandleFunc("/"+version.API()+"/manifest", GzipHandler(ServeManifest))
	http.HandleFunc("/"+version.API()+"/sync", GzipHandler(ServeSync))
	http.HandleFunc("/version", GzipHandler(ServeServerVer))
}
