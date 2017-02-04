//Package node provides the node side logic of the node.
package node

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
	"github.com/tywkeene/autobd/client"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/utils"
	"github.com/tywkeene/autobd/version"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type Node struct {
	Servers map[string]*client.Client
	UUID    string
	Synced  bool
	Config  options.NodeConf
}

var localNode *Node
var nodeUseragent string = "Autobd-node/" + version.GetNodeVersion()

func newNode(config options.NodeConf) *Node {
	servers := make(map[string]*client.Client, 0)
	for _, url := range config.Servers {
		servers[url] = client.NewClient(url)
	}
	return &Node{servers, "", false, config}
}

func InitNode(config options.NodeConf) *Node {
	node := newNode(config)
	//Check to see if we already have a UUID stored in a file, if not, generate one and
	//write it to node.Config.UUIDPath
	if _, err := os.Stat(config.UUIDPath); os.IsNotExist(err) {
		node.UUID = uuid.NewV4().String()
		node.WriteNodeUUID()
		log.Infof("Generated and wrote node UUID (%s) to (%s) ", node.UUID, node.Config.UUIDPath)
	} else {
		node.ReadNodeUUID()
		log.Infof("Read node UUID (%s) from (%s) ", node.UUID, node.Config.UUIDPath)
	}
	return node
}

func (node *Node) WriteNodeUUID() error {
	outfile, err := os.Create(node.Config.UUIDPath)
	if err != nil {
		return err
	}
	defer outfile.Close()
	serial, err := json.MarshalIndent(&node.UUID, " ", " ")
	if err != nil {
		return err
	}
	_, err = outfile.WriteString(string(serial))
	return err
}

func (node *Node) ReadNodeUUID() error {
	if _, err := os.Stat(node.Config.UUIDPath); err != nil {
		return err
	}
	serial, err := ioutil.ReadFile(node.Config.UUIDPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(serial, &node.UUID)
}

func (node *Node) validateServerVersion(remote *version.VersionInfo) error {
	if version.GetAPIVersion() != remote.APIVersion {
		return fmt.Errorf("Mismatched version with server. Server: %s Local: %s",
			remote.APIVersion, version.GetAPIVersion())
	}
	remoteMajor := strings.Split(remote.APIVersion, ".")[0]
	if version.GetMajor() != remoteMajor {
		return fmt.Errorf("Mismatched API version with server. Server: %s Local: %s",
			remoteMajor, version.GetMajor())
	}
	return nil
}

func (node *Node) StartHeart() {
	go func(config options.NodeConf) {
		interval, _ := time.ParseDuration(config.HeartbeatInterval)
		log.Info("Started heartbeat, updating every ", interval)
		for {
			time.Sleep(interval)
			for _, server := range node.Servers {
				if server.Online == false {
					continue
				}
				_, err := server.SendHeartbeat(node.UUID, node.Synced, nodeUseragent)
				if utils.HandleError("node/StartHeart()", err, utils.ErrorActionErr) == true {
					server.MissedBeats++
					if server.MissedBeats == node.Config.MaxMissedBeats {
						server.Online = false
						log.Error(server.Address + " has missed max heartbeats, ignoring")
					}
				}
			}
		}
	}(node.Config)
}

func (node *Node) CountOnlineServers() int {
	var count int = 0
	for _, server := range node.Servers {
		if server.Online == true {
			count++
		}
	}
	return count
}

func (node *Node) ValidateAndIdentifyWithServers() error {
	for _, server := range node.Servers {
		serial, err := server.RequestVersion()
		if err != nil {
			return err
		}
		var remoteVer *version.VersionInfo
		if err := json.Unmarshal(serial, &remoteVer); err != nil {
			return err
		}

		if options.Config.NodeConfig.IgnoreVersionMismatch == false {
			if err := node.validateServerVersion(remoteVer); err != nil {
				return err
			}
		}
		_, err = server.IdentifyWithServer(node.UUID, nodeUseragent)
		if utils.HandleError("node/ValidateAndIdentifyWithServers", err, utils.ErrorActionErr) == true {
			continue
		}
	}
	node.StartHeart()
	return nil
}

func CompareDirs(local map[string]*index.Index, remote map[string]*index.Index) []*index.Index {
	need := make([]*index.Index, 0)
	for objName, object := range remote {
		_, existsLocally := local[object.Name] //Does it exist on the node?

		//If it doesn't exist on the node at all, we obviously need it
		if existsLocally == false {
			need = append(need, remote[objName])
			continue
		}

		// If it does, and it's a directory, and it has children
		if existsLocally == true && object.IsDir == true && object.Files != nil {
			dirNeed := CompareDirs(local[objName].Files, object.Files) //Scan the children
			need = append(need, dirNeed...)
			continue
		}

		//If it isn't a directory and doesn't exist
		if existsLocally == false && object.IsDir == false {
			need = append(need, remote[objName])
			continue
		}

		//If it is a file and does exist, compare checksums
		if existsLocally == true && object.IsDir == false {
			if local[objName].Checksum != remote[objName].Checksum {
				log.Info("Checksum mismatch:", objName)
				need = append(need, remote[objName])
				continue
			}
		}
	}
	return need
}

//Compare a local and remote index, return a slice of needed indexes (or nil)
func (node *Node) CompareIndex(target string, uuid string, userAgent string, client *client.Client) ([]*index.Index, error) {
	serial, err := client.RequestIndex(target, uuid, userAgent)
	if err != nil {
		return nil, err
	}
	var remoteIndex map[string]*index.Index
	if err := json.Unmarshal(serial, &remoteIndex); err != nil {
		return nil, err
	}
	localIndex, err := index.GetIndex("/")
	if err != nil {
		return nil, err
	}
	need := CompareDirs(localIndex, remoteIndex)
	return need, nil
}

func (node *Node) SyncUp(need []*index.Index, s *client.Client) {
	for _, object := range need {
		log.Printf("Need %s from %s", object.Name, s.Address)
		if object.IsDir == true {
			err := s.RequestSyncDir(object.Name, node.UUID, nodeUseragent)
			if utils.HandleError("node/SyncUp()", err, utils.ErrorActionInfo) == true {
				continue
			}
		} else if object.IsDir == false {
			err := s.RequestSyncFile(object.Name, node.UUID, nodeUseragent)
			if err != nil {
				//EOF just means the sync is finished, don't log an error
				utils.HandleError("node/SyncUp()", err, utils.ErrorActionInfo)
				continue
			}
		}
	}
}

func (node *Node) UpdateLoop() error {
	err := node.ValidateAndIdentifyWithServers()
	utils.HandlePanic("node/UpdateLoop", err)
	log.Printf("Running as a node. Updating every %s with %s",
		node.Config.UpdateInterval, node.Config.Servers)

	updateInterval, err := time.ParseDuration(node.Config.UpdateInterval)
	utils.HandlePanic("node/UpdateLoop()", err)
	for {
		time.Sleep(updateInterval)
		if node.CountOnlineServers() == 0 {
			utils.HandlePanic("node/UpdateLoop()", fmt.Errorf("No servers online, dying"))
		}
		for _, server := range node.Servers {
			if server.Online == false {
				log.Info("Skipping offline server: ", server.Address)
				continue
			}
			need, err := node.CompareIndex(node.Config.TargetDirectory, node.UUID, nodeUseragent, server)
			if utils.HandleError("node/UpdateLoop()", err, utils.ErrorActionWarn) == true {
				continue
			}

			if len(need) == 0 {
				node.Synced = true
				continue
			}
			node.SyncUp(need, server)
		}
	}
}
