//Package api implements the endpoints and utilities necessary to present
//a consistent API to autobd-nodes

package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/packing"
	"github.com/tywkeene/autobd/utils"
	"github.com/tywkeene/autobd/version"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NodeHeartbeat struct {
	Synced string `json"synced"`
	UUID   string `json:"UUID"`
}

type NodeMetadata struct {
	Version string `json:"version"`
	UUID    string `json:"UUID"`
}

type Node struct {
	Address    string        `json:"address"`     //Address of the node
	LastOnline string        `json:"last_online"` //Timestamp of when the node last sent a heartbeat
	IsOnline   bool          `json:"is_online"`   //Is the node currently online?
	Synced     bool          `json:"synced"`      //Is the node synced with this server?
	Meta       *NodeMetadata `json:"metadata"`    //Node Version, UUID and other misc. information about this node
}

type NodeList map[string]*Node

//Currently registered nodes indexed by uuid
var CurrentNodes NodeList

// For synchronized access to CurrentNodes
var lock = sync.RWMutex{}

func LogHttp(r *http.Request) {
	log.Printf("%s %s %s %s", r.Method, r.URL, r.RemoteAddr, r.UserAgent())
}

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

//GetQueryValue() takes a name of a key:value pair to fetch from a URL encoded query,
//a http.ResponseWriter 'w', and a http.Request 'r'. In the event that an error is encountered
//the error will be returned to the client via logging facilities that use 'w' and 'r'
func GetQueryValue(name string, w http.ResponseWriter, r *http.Request) string {
	errHandle := utils.NewHttpErrorHandle("api/GetQueryValue()", w, r)
	query, err := url.ParseQuery(r.URL.RawQuery)
	if errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr) == true {
		return ""
	}
	return query.Get(name)
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

	uuid := GetQueryValue("uuid", w, r)
	if validateNode(uuid) == false || uuid == "" {
		errHandle.Handle(fmt.Errorf("Invalid node UUID"),
			http.StatusUnauthorized, utils.ErrorActionErr)
		return
	}

	dir := GetQueryValue("dir", w, r)
	if dir == "" {
		errHandle.Handle(fmt.Errorf("Must specify directory"),
			http.StatusBadRequest, utils.ErrorActionErr)
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
	uuid := GetQueryValue("uuid", w, r)
	if validateNode(uuid) == false {
		errHandle.Handle(fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized, utils.ErrorActionErr)
		return
	}
	grab := GetQueryValue("grab", w, r)
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
	updateNodeStatus(uuid, true, true)
}

//Add a node to the CurrentNodes map synchronously
func GetNodeByUUID(uuid string) *Node {
	lock.RLock()
	defer lock.RUnlock()
	if uuid == "" || CurrentNodes == nil {
		return nil
	}
	return CurrentNodes[uuid]
}

//Get a node from the CurrentNodes map synchronously
func AddNode(uuid string, node *Node) {
	lock.RLock()
	defer lock.RUnlock()

	if CurrentNodes == nil {
		CurrentNodes = make(map[string]*Node)
	}
	CurrentNodes[uuid] = node
}

//Update the online status and timestamp of a node by uuid
func updateNodeStatus(uuid string, online bool, synced bool) {
	node := GetNodeByUUID(uuid)
	if online == true {
		node.LastOnline = time.Now().Format(time.RFC850)
	}
	node.IsOnline = online
	node.Synced = synced
}

//Validate a node uuid
func validateNode(uuid string) bool {
	if node := GetNodeByUUID(uuid); node == nil {
		return false
	}
	return true
}

func ReadNodeList(path string) error {
	serial, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(serial, &CurrentNodes); err != nil {
		return err
	}
	return nil
}

func WriteNodeList(path string) error {
	outfile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outfile.Close()
	serial, err := json.MarshalIndent(&CurrentNodes, " ", " ")
	if err != nil {
		return err
	}
	_, err = outfile.WriteString(string(serial))
	return err
}

//ListNodes() is the http handler for the "/nodes" API endpoint
//It returns the CurrentNodes map encoded in json
func ListNodes(w http.ResponseWriter, r *http.Request) {
	defer utils.TimeTrack(time.Now(), "api/ListNodes()")
	errHandle := utils.NewHttpErrorHandle("api/ListNodes()", w, r)
	LogHttp(r)
	uuid := GetQueryValue("uuid", w, r)
	if validateNode(uuid) == false {
		errHandle.Handle(fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized, utils.ErrorActionErr)
		return
	}
	lock.RLock()
	defer lock.RUnlock()
	serial, _ := json.MarshalIndent(&CurrentNodes, " ", " ")
	io.WriteString(w, string(serial))
}

//StartHeartBeatTracker() is go routine that will periodically update the status of all
//nodes currently registered with the server
func StartHeartBeatTracker() {
	log.Infof("Updating nodes status every %s", options.Config.HeartBeatTrackInterval)

	interval, err := time.ParseDuration(options.Config.HeartBeatTrackInterval)
	utils.HandlePanic(err)

	cutoff, err := time.ParseDuration(options.Config.HeartBeatOffline)
	utils.HandlePanic(err)

	for {
		time.Sleep(interval)
		lock.RLock()
		for uuid, node := range CurrentNodes {
			then, err := time.Parse(time.RFC850, node.LastOnline)
			utils.HandlePanic(err)
			duration := time.Since(then)
			if duration > cutoff && node.IsOnline == true {
				log.Warnf("Node %s has not checked in since %s ago, marking offline", uuid, duration)
				updateNodeStatus(uuid, false, node.Synced)
			}
		}
		lock.RUnlock()
	}
}

func GetPostData(r *http.Request, data interface{}) error {
	serial, err := ioutil.ReadAll(r.Body)
	if err == nil {
		return err
	}
	if err := json.Unmarshal(serial, &data); err != nil {
		return err
	}
	return nil
}

//Identify() is the http handler for the "/identify" API endpoint
//It takes a node UUID and node version as json encoded strings
//The node is added to the CurrentNodes map, with the RFC850 timestamp
func Identify(w http.ResponseWriter, r *http.Request) {
	defer utils.TimeTrack(time.Now(), "api/Identify()")
	errHandle := utils.NewHttpErrorHandle("api/Identify()", w, r)
	LogHttp(r)

	var metaData *NodeMetadata
	err := GetPostData(r, &metaData)
	if err != nil || metaData == nil {
		errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr)
		return
	}

	lock.RLock()
	defer lock.RUnlock()
	//Initialize the node list and start the heartbeat tracker
	if CurrentNodes == nil {
		CurrentNodes = make(map[string]*Node)
		go StartHeartBeatTracker()
	}

	//Handle to see if this node is already online
	if validateNode(metaData.UUID) == true {
		node := GetNodeByUUID(metaData.UUID)
		if node.IsOnline == false {
			node.IsOnline = true
		}
		log.Printf("Node (%s) came back online", metaData.UUID)
	} else {
		//Otherwise it's new, so add it to the list
		AddNode(metaData.UUID,
			&Node{
				Address:    r.RemoteAddr,
				LastOnline: time.Now().Format(time.RFC850),
				IsOnline:   true,
				Synced:     false,
				Meta:       metaData,
			})
		log.Printf("New node UUID:(%s) Address:[%s] Version:%s",
			metaData.UUID, r.RemoteAddr, metaData.Version)
	}
	WriteNodeList(options.Config.NodeListFile)
}

//HeartBeat() is the http handler for the "/heartbeat" API endpoint
//Nodes will request this every config.HeartbeatInterval and the server will update
//their respective online timestamp
func HeartBeat(w http.ResponseWriter, r *http.Request) {
	defer utils.TimeTrack(time.Now(), "api/HeartBeat()")
	errHandle := utils.NewHttpErrorHandle("api/HeartBeat()", w, r)
	LogHttp(r)
	var heartbeat *NodeHeartbeat
	err := GetPostData(r, &heartbeat)
	if err != nil || heartbeat == nil {
		errHandle.Handle(err, http.StatusInternalServerError, utils.ErrorActionErr)
		return
	}
	if validateNode(heartbeat.UUID) == false {
		errHandle.Handle(fmt.Errorf("Invalid node UUID"), http.StatusUnauthorized, utils.ErrorActionErr)
		return
	}
	synced, _ := strconv.ParseBool(heartbeat.Synced)
	updateNodeStatus(heartbeat.UUID, true, synced)
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
