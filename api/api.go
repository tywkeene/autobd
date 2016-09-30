//Package api implements the endpoints and utilities necessary to present
//a consistent API to autobd-nodes

package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/logging"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/packing"
	"github.com/tywkeene/autobd/version"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Node struct {
	Address string
	Version string
	Online  string
	Synced  bool
}

var CurrentNodes map[string]*Node

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

//ServeIndex() is the http handler for the "/index" API endpoint.
//It takes the requested directory passed as a url parameter "dir" i.e "/index?dir=/"
//
//It will then generate a index by calling api.GetQueryValue(), then writes it to the client as a
//map[string]*index.Index encoded in json
func ServeIndex(w http.ResponseWriter, r *http.Request) {
	logging.LogHttp(r)

	uuid := GetQueryValue("uuid", w, r)
	if validateNode(uuid) == false {
		logging.LogHttpErr(w, r, fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized)
		return
	}
	updateNodeOnline(uuid)

	dir := GetQueryValue("dir", w, r)
	if dir == "" {
		log.Println("No directory defined")
		logging.LogHttpErr(w, r, fmt.Errorf("Must define directory"), http.StatusInternalServerError)
		return
	}
	dirIndex, err := index.GetIndex(dir)
	if err != nil {
		log.Println(err)
		logging.LogHttpErr(w, r, fmt.Errorf("Error getting index"), http.StatusInternalServerError)
		return
	}
	serial, _ := json.MarshalIndent(&dirIndex, "  ", "  ")
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
	uuid := GetQueryValue("uuid", w, r)
	if validateNode(uuid) == false {
		logging.LogHttpErr(w, r, fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized)
		return
	}
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
	updateNodeSynced(uuid, true)
	updateNodeOnline(uuid)
}

//Identify() is the http handler for the "/identify" API endpoint
//It takes a node UUID and node version as json encoded strings
//The node is added to the CurrentNodes map, with the RFC850 timestamp
func Identify(w http.ResponseWriter, r *http.Request) {
	logging.LogHttp(r)
	uuid := GetQueryValue("uuid", w, r)
	version := GetQueryValue("version", w, r)
	log.Printf("New node [UUID: %s Address: %s Version: %s]", uuid, r.RemoteAddr, version)
	if CurrentNodes == nil {
		CurrentNodes = make(map[string]*Node)
		log.Println("Initialized node list")
	}
	CurrentNodes[uuid] = &Node{r.RemoteAddr, version, time.Now().Format(time.RFC850), false}
}

func updateNodeSynced(uuid string, val bool) {
	node := CurrentNodes[uuid]
	node.Synced = val
}

func updateNodeOnline(address string) {
	node := CurrentNodes[address]
	node.Online = time.Now().Format(time.RFC850)
}

func validateNode(uuid string) bool {
	_, ok := CurrentNodes[uuid]
	return ok
}

//ListNodes() is the http handler for the "/nodes" API endpoint
//It returns the CurrentNodes map encoded in json
func ListNodes(w http.ResponseWriter, r *http.Request) {
	logging.LogHttp(r)
	serial, _ := json.MarshalIndent(&CurrentNodes, " ", " ")
	io.WriteString(w, string(serial))
}

//HeartBeat() is the http handler for the "/heartbeat" API endpoint
//Nodes will request this every config.HeartbeatInterval and the server will update
//their respective online timestamp
func HeartBeat(w http.ResponseWriter, r *http.Request) {
	logging.LogHttp(r)
	uuid := GetQueryValue("uuid", w, r)
	if validateNode(uuid) == false {
		logging.LogHttpErr(w, r, fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized)
		return
	}
	updateNodeOnline(uuid)
}

func SetupRoutes() {
	http.HandleFunc("/v"+version.Major()+"/index", GzipHandler(ServeIndex))
	http.HandleFunc("/v"+version.Major()+"/sync", GzipHandler(ServeSync))
	http.HandleFunc("/v"+version.Major()+"/identify", GzipHandler(Identify))
	if options.Config.NodeEndpoint == true {
		http.HandleFunc("/v"+version.Major()+"/nodes", GzipHandler(ListNodes))
	}
	http.HandleFunc("/v"+version.Major()+"/heartbeat", GzipHandler(HeartBeat))
	http.HandleFunc("/version", GzipHandler(ServeServerVer))
}
