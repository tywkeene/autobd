package main

import (
	"encoding/json"
	"fmt"
	"github.com/SaviorPhoenix/autobd/compression"
	"github.com/SaviorPhoenix/autobd/helpers"
	"github.com/SaviorPhoenix/autobd/options"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"
)

var (
	apiVersion string = "v0"
	version    string = "0.1"
	commit     string
)

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
	w.Header().Set("Server", "Autobd v"+version)
	io.WriteString(w, string(serial))
}

func ServeVersion(w http.ResponseWriter, r *http.Request) {
	type versionInfo struct {
		Ver     string `json:"server"`
		Api     string `json:"api"`
		Commit  string `json:"commit"`
		Comment string `json:"comment"`
	}
	serialVer, _ := json.MarshalIndent(&versionInfo{version, apiVersion, commit,
		"API not intended for human consumption"}, "  ", "  ")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+version)
	io.WriteString(w, string(serialVer))
}

func ServeSync(w http.ResponseWriter, r *http.Request) {
	helpers.LogHttp(r)
	grab := helpers.GetQueryValue("grab", w, r)
	if grab == "" {
		return
	}
	http.ServeFile(w, r, grab)
}

func versionInfo() {
	if commit == "" {
		commit = "unknown"
	}
	fmt.Printf("Autobd version %s (API %s) (git commit %s)\n", version, apiVersion, commit)
}

func setupRoutes() {
	http.HandleFunc("/"+apiVersion+"/manifest", compression.MakeGzipHandler(ServeManifest))
	http.HandleFunc("/"+apiVersion+"/sync", compression.MakeGzipHandler(ServeSync))
	http.HandleFunc("/version", compression.MakeGzipHandler(ServeVersion))
}

func init() {
	versionInfo()
	options.GetOptions()
	setupRoutes()
}

func main() {
	if err := syscall.Chroot(*options.Flags.Root); err != nil {
		panic("chroot: " + err.Error())
	}
	if err := os.Chdir(*options.Flags.Root); err != nil {
		panic(err)
	}
	log.Printf("Serving '%s' on port %s", *options.Flags.Root, *options.Flags.ApiPort)
	log.Panic(http.ListenAndServe(":"+*options.Flags.ApiPort, nil))
}
