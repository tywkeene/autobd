//Package node provides the node side logic of the node.
package node

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
	"github.com/tywkeene/autobd/connection"
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
	Servers map[string]*connection.Connection
	UUID    string
	Config  options.NodeConf
}

var localNode *Node

func newNode(config options.NodeConf) *Node {
	userAgent := "Autobd-node/" + version.GetVersion()
	servers := make(map[string]*connection.Connection, 0)
	for _, url := range config.Servers {
		servers[url] = connection.NewConnection(url, userAgent)
	}
	return &Node{Servers: servers, UUID: "", Config: config}
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
	remoteMajor := strings.Split(remote.Version, ".")[0]
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
				_, err := server.SendHeartbeat(node.UUID)
				if utils.HandleError(err, utils.ErrorActionErr) == true {
					server.MissedBeats++
					if server.MissedBeats == node.Config.MaxMissedBeats {
						server.SetOnline(false)
						server.SetSynced(false)
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

func (node *Node) Identify() error {
	for _, server := range node.Servers {
		serial, err := server.RequestVersion()
		if err != nil {
			return err
		}
		var remoteVer *version.VersionInfo
		if err := json.Unmarshal(serial, &remoteVer); err != nil {
			return err
		}

		if err := node.validateServerVersion(remoteVer); err != nil {
			if options.Config.NodeConfig.IgnoreVersionMismatch == false {
				log.Warnf("Server (%s) is running a different API version. Some functionality may be broken!\n",
					server.Address)
				return err
			}
		}
		_, err = server.IdentifyWithServer(version.GetVersion(), node.UUID, options.Config.NodeConfig.TargetDirectory)
		if utils.HandleError(err, utils.ErrorActionErr) == true {
			continue
		}
	}
	node.StartHeart()
	return nil
}

func CompareDirs(local map[string]*index.Index, remote map[string]*index.Index) []*index.Index {
	need := make([]*index.Index, 0)
	for objName, remoteObject := range remote {
		_, existsLocally := local[remoteObject.Name] //Does it exist on the node?
		//If it doesn't exist on the node at all, we obviously need it
		if existsLocally == false {
			need = append(need, remoteObject)
			continue
		}
		// If it does, and it's a directory, and it has children
		if existsLocally == true && remoteObject.IsDir == true && remoteObject.Files != nil {
			dirNeed := CompareDirs(local[objName].Files, remoteObject.Files) //Scan the children
			need = append(need, dirNeed...)
			continue
		}
		//If it is a file and does exist, compare checksums
		if existsLocally == true && remoteObject.IsDir == false {
			if local[objName].Checksum != remoteObject.Checksum {
				log.Info("Checksum mismatch:", objName)
				need = append(need, remoteObject)
				continue
			}
		}
	}
	return need
}

//Compare a local and remote index, return a slice of needed indexes (or nil)
func (node *Node) CompareIndex(target string, server *connection.Connection) ([]*index.Index, error) {
	serial, err := server.RequestIndex(target, node.UUID)
	if utils.HandleError(err, utils.ErrorActionErr) == true {
		return nil, err
	}
	var remoteIndex map[string]*index.Index
	if err := json.Unmarshal(serial, &remoteIndex); err != nil {
		if utils.HandleError(err, utils.ErrorActionErr) == true {
			return nil, err
		}
	}
	if _, err := os.Stat(target); os.IsNotExist(err) {
		os.Mkdir(target, 0755)
	}
	localIndex, err := index.GetIndex(target)
	if err != nil {
		return nil, err
	}
	need := CompareDirs(localIndex, remoteIndex)
	return need, nil
}

func (node *Node) IsSynced() bool {
	for _, server := range node.Servers {
		if server.Synced == false {
			return false
		}
	}
	return true
}

func (node *Node) Sync(server *connection.Connection) error {
	need, err := node.CompareIndex(node.Config.TargetDirectory, server)
	if err != nil {
		return err
	}
	if len(need) > 0 {
		server.SetSynced(false)
		for _, object := range need {
			log.Printf("%s -> Need:%s", server.Address, object.Name)
			if object.IsDir == true {
				err := server.RequestSyncDir(object.Name, node.UUID)
				if utils.HandleError(err, utils.ErrorActionInfo) == true {
					continue
				}
			} else if object.IsDir == false {
				err := server.RequestSyncFile(object.Name, node.UUID)
				if err != nil {
					//EOF just means the sync is finished, don't log an error
					utils.HandleError(err, utils.ErrorActionInfo)
					continue
				}
			}
		}
	} else {
		server.SetSynced(true)
	}
	return nil
}

func (node *Node) UpdateLoop() error {
	err := node.Identify()
	utils.HandlePanic(err)

	log.Printf("Running as a node. Updating every %s with %s",
		node.Config.UpdateInterval, node.Config.Servers)

	updateInterval, err := time.ParseDuration(node.Config.UpdateInterval)
	utils.HandlePanic(err)
	for {
		time.Sleep(updateInterval)
		if node.CountOnlineServers() == 0 {
			utils.HandlePanic(fmt.Errorf("No servers online, dying"))
		}
		for _, server := range node.Servers {
			if server.Online == false {
				log.Info("Skipping offline server: ", server.Address)
				continue
			}
			err := node.Sync(server)
			if utils.HandleError(err, utils.ErrorActionWarn) == true {
				break
			}
		}
	}
}
