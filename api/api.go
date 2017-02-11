//Package api implements the endpoints and utilities necessary to present
//a consistent API to autobd-nodes

package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/nodelist"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/packing"
	"github.com/tywkeene/autobd/utils"
	"github.com/tywkeene/autobd/version"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

//Handle and make sure the client wants or can handle gzip, and replace the writer if it
//can, if not, simply use the normal http.ResponseWriter
func GzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "application/x-gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "application/x-gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func LogHttp(r *http.Request) {
	log.Printf("%s %s %s %s", r.Method, r.URL, r.RemoteAddr, r.UserAgent())
}

//ServeIndex() is the http handler for the "/index" API endpoint.
//It takes the requested directory passed as a url parameter "dir" i.e "/index?dir=/"
//
//It will then generate a index by calling api.GetIndex(), then writes it to the client as a
//map[string]*index.Index encoded in json
func ServeIndex(w http.ResponseWriter, r *http.Request) {
	defer utils.TimeTrack(time.Now(), "api/ServeIndex()")
	errHandle := utils.NewHttpErrorHandle("api/ServeIndex()", w, r)
	LogHttp(r)

	uuid, err := nodelist.GetQueryValue("uuid", w, r)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) == true {
		return
	}
	if nodelist.ValidateNode(uuid) == false {
		errHandle.Handle(fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized, utils.ErrorActionErr)
		return
	}

	dir, err := nodelist.GetQueryValue("dir", w, r)
	if dir == "" {
		errHandle.Handle(fmt.Errorf("Must specify directory"), http.StatusBadRequest, utils.ErrorActionErr)
		return
	}
	dirIndex, err := index.GetIndex(dir)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) == true {
		return
	}
	serial, _ := json.MarshalIndent(&dirIndex, "  ", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+version.GetAPIVersion())
	io.WriteString(w, string(serial))
}

//ServeServerVer() is the http handler for the "/version" http API endpoint.
//It writes the json encoded struct version.VersionInfo to the client
func ServeServerVer(w http.ResponseWriter, r *http.Request) {
	defer utils.TimeTrack(time.Now(), "api/ServeServerVer()")
	LogHttp(r)
	serialVer := version.JSON()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "Autobd v"+version.GetAPIVersion())
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
	defer utils.TimeTrack(time.Now(), "api/ServeSync()")
	errHandle := utils.NewHttpErrorHandle("api/ServeSync()", w, r)
	LogHttp(r)
	uuid, err := nodelist.GetQueryValue("uuid", w, r)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) == true {
		return
	}
	if nodelist.ValidateNode(uuid) == false {
		errHandle.Handle(fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized, utils.ErrorActionErr)
		return
	}
	grab, err := nodelist.GetQueryValue("grab", w, r)
	if errHandle.Handle(err, http.StatusBadRequest, utils.ErrorActionErr) == true {
		return
	}
	if grab == "" {
		return
	}
	fd, err := os.Open(grab)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) == true {
		return
	}
	defer fd.Close()
	info, err := fd.Stat()
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) == true {
		return
	}
	if info.IsDir() == true {
		err := packing.PackDir(grab, w)
		if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) == true {
			return
		}
		w.Header().Set("Content-Type", "application/x-tar")
		return
	}
	http.ServeContent(w, r, grab, info.ModTime(), fd)
	nodelist.UpdateNodeStatus(uuid, true, true)
}

//ListNodes() is the http handler for the "/nodes" API endpoint
//It returns the CurrentNodes map encoded in json
func ListNodes(w http.ResponseWriter, r *http.Request) {
	defer utils.TimeTrack(time.Now(), "api/ListNodes()")
	errHandle := utils.NewHttpErrorHandle("api/ListNodes()", w, r)
	LogHttp(r)
	uuid, err := nodelist.GetQueryValue("uuid", w, r)
	if errHandle.Handle(err, http.StatusUnauthorized, utils.ErrorActionErr) == true {
		return
	}
	if nodelist.ValidateNode(uuid) == false {
		errHandle.Handle(fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized, utils.ErrorActionErr)
		return
	}
	nodeList := nodelist.GetNodelistJson()
	io.WriteString(w, string(nodeList))
}

//StartHeartBeatTracker() is go routine that will periodically update the status of all
//nodes currently registered with the server
func StartHeartBeatTracker() {
	log.Infof("Updating nodes status every %s", options.Config.HeartBeatTrackInterval)

	interval, err := time.ParseDuration(options.Config.HeartBeatTrackInterval)
	utils.HandlePanic(err)

	for {
		time.Sleep(interval)
		nodelist.UpdateNodeList()
	}
}

//Identify() is the http handler for the "/identify" API endpoint
//It takes a node UUID and node version as json encoded strings
//The node is added to the CurrentNodes map, with the RFC850 timestamp
func Identify(w http.ResponseWriter, r *http.Request) {
	defer utils.TimeTrack(time.Now(), "api/Identify()")
	errHandle := utils.NewHttpErrorHandle("api/Identify()", w, r)
	LogHttp(r)

	serial, err := ioutil.ReadAll(r.Body)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) {
		return
	}
	var metaData *nodelist.NodeMetadata
	err = json.Unmarshal(serial, &metaData)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) {
		return
	}

	nodelist.InitializeNodeList()

	//Handle to see if this node is already tracked
	if nodelist.ValidateNode(metaData.UUID) == true {
		node := nodelist.GetNodeByUUID(metaData.UUID)
		if node.IsOnline == false {
			node.IsOnline = true
		}
		log.Printf("Node (%s) came back online", metaData.UUID)
	} else {
		//Otherwise it's new, so add it to the list
		nodelist.AddNode(metaData.UUID,
			&nodelist.Node{
				Address:    r.RemoteAddr,
				LastOnline: time.Now().Format(time.RFC850),
				IsOnline:   true,
				Synced:     false,
				Meta:       metaData,
			})
		log.Printf("New node UUID:(%s) Address:[%s] Version:%s",
			metaData.UUID, r.RemoteAddr, metaData.Version)
		nodelist.WriteNodeList(options.Config.NodeListFile)
	}
}

//HeartBeat() is the http handler for the "/heartbeat" API endpoint
//Nodes will request this every config.HeartbeatInterval and the server will update
//their respective online timestamp
func HeartBeat(w http.ResponseWriter, r *http.Request) {
	defer utils.TimeTrack(time.Now(), "api/HeartBeat()")
	errHandle := utils.NewHttpErrorHandle("api/HeartBeat()", w, r)
	LogHttp(r)

	serial, err := ioutil.ReadAll(r.Body)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) {
		return
	}
	var heartbeat *nodelist.NodeHeartbeat
	err = json.Unmarshal(serial, &heartbeat)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) {
		return
	}
	if nodelist.ValidateNode(heartbeat.UUID) == false {
		errHandle.Handle(fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized, utils.ErrorActionErr)
		return
	}
	synced, _ := strconv.ParseBool(heartbeat.Synced)
	nodelist.UpdateNodeStatus(heartbeat.UUID, true, synced)
}

func SetupRoutes() {
	http.HandleFunc("/v"+version.GetMajor()+"/index", GzipHandler(ServeIndex))
	http.HandleFunc("/v"+version.GetMajor()+"/sync", GzipHandler(ServeSync))
	http.HandleFunc("/v"+version.GetMajor()+"/identify", GzipHandler(Identify))
	if options.Config.NodeEndpoint == true {
		http.HandleFunc("/v"+version.GetMajor()+"/nodes", GzipHandler(ListNodes))
	}
	http.HandleFunc("/v"+version.GetMajor()+"/heartbeat", GzipHandler(HeartBeat))
	http.HandleFunc("/version", GzipHandler(ServeServerVer))
}
