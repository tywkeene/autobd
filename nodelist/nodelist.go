package nodelist

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/utils"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type NodeHeartbeat struct {
	Synced string `json:"synced"`
	UUID   string `json:"UUID"`
}

type NodeMetadata struct {
	Version string `json:"version"`
	UUID    string `json:"UUID"`
	Target  string `json:"node_target_directory"`
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

func (node *Node) ShortUUID() string {
	return node.Meta.UUID[:8]
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
func UpdateNodeStatus(uuid string, online bool, synced bool) {
	node := GetNodeByUUID(uuid)
	if online == true {
		node.LastOnline = time.Now().Format(time.RFC850)
	}
	node.IsOnline = online
	node.Synced = synced
}

//Validate a node uuid
func ValidateNode(uuid string) bool {
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

func UpdateNodeList() {
	cutoff, err := time.ParseDuration(options.Config.HeartBeatOffline)
	utils.HandlePanic(err)

	lock.RLock()
	defer lock.RUnlock()
	for uuid, node := range CurrentNodes {
		then, err := time.Parse(time.RFC850, node.LastOnline)
		utils.HandlePanic(err)
		duration := time.Since(then)
		if duration > cutoff && node.IsOnline == true {
			log.Warnf("Node %s has not checked in since %s ago, marking offline", uuid, duration)
			UpdateNodeStatus(uuid, false, node.Synced)
			err := WriteNodeList(options.Config.NodeListFile)
			utils.HandleError(err, utils.ErrorActionErr)
		}
	}
}

func GetNodelistJson() []byte {
	lock.RLock()
	defer lock.RUnlock()
	serial, err := json.MarshalIndent(&CurrentNodes, " ", " ")
	if utils.HandleError(err, utils.ErrorActionErr) == true {
		return nil
	}
	return serial
}

func InitializeNodeList() {
	lock.RLock()
	defer lock.RUnlock()
	//Initialize the node list and start the heartbeat tracker
	if CurrentNodes == nil {
		CurrentNodes = make(map[string]*Node)
	}
}
