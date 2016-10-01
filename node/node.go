//Package node provides the node side logic of the node.
package node

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/server"
	"github.com/tywkeene/autobd/version"
	"strings"
	"time"
)

type Node struct {
	Servers map[string]*server.Server
	UUID    string
	Synced  bool
	Config  options.NodeConf
}

var localNode *Node

func newNode(config options.NodeConf) *Node {
	servers := make(map[string]*server.Server, 0)
	for _, url := range config.Servers {
		servers[url] = server.NewServer(url)
	}
	UUID := uuid.NewV4().String()
	log.Info("Generated node UUID: ", UUID)
	return &Node{servers, UUID, false, config}
}

func InitNode(config options.NodeConf) *Node {
	node := newNode(config)
	return node
}

func (node *Node) validateServerVersion(remote *version.VersionInfo) error {
	if version.Server() != remote.ServerVer {
		return fmt.Errorf("Mismatched version with server. Server: %s Local: %s",
			remote.ServerVer, version.Server())
	}
	remoteMajor := strings.Split(remote.ServerVer, ".")[0]
	if version.Major() != remoteMajor {
		return fmt.Errorf("Mismatched API version with server. Server: %s Local: %s",
			remoteMajor, version.Major())
	}
	return nil
}

func (node *Node) StartHeart() {
	go func(config options.NodeConf) {
		interval, _ := time.ParseDuration(config.HeartbeatInterval)
		for {
			time.Sleep(interval)
			for _, server := range node.Servers {
				if server.Online == false {
					continue
				}
				_, err := server.SendHeartbeat(node.UUID, node.Synced)
				if err != nil {
					log.Error(err)
					server.MissedBeats++
					if server.MissedBeats == node.Config.MaxMissedBeats {
						server.Online = false
						log.Error(server.Address + " has missed max beats, ignoring")
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

func (node *Node) ValidateAndIdentifyServers() error {
	for _, server := range node.Servers {
		remoteVer, err := server.RequestVersion()
		if remoteVer == nil || err != nil {
			return err
		}
		if options.Config.NodeConfig.IgnoreVersionMismatch == false {
			if err := node.validateServerVersion(remoteVer); err != nil {
				log.Error(err)
				return err
			}
		}
		_, err = server.IdentifyWithServer(node.UUID)
		if err != nil {
			log.Error(err)
			continue
		}
	}
	node.StartHeart()
	return nil
}

func (node *Node) UpdateLoop() error {
	if err := node.ValidateAndIdentifyServers(); err != nil {
		return err
	}
	log.Printf("Running as a node. Updating every %s with %s",
		node.Config.UpdateInterval, node.Config.Servers)

	updateInterval, err := time.ParseDuration(node.Config.UpdateInterval)
	if err != nil {
		return err
	}
	for {
		time.Sleep(updateInterval)
		if node.CountOnlineServers() == 0 {
			log.Panic("No servers online, dying")
		}
		for _, server := range node.Servers {
			if server.Online == false {
				log.Info("Skipping offline server: ", server.Address)
				continue
			}
			log.Info("Updating with ", server.Address)
			need, err := server.CompareIndex(node.Config.TargetDirectory, node.UUID)
			if err != nil {
				log.Error(err)
				continue
			}

			if len(need) == 0 {
				log.Info("In sync with ", server.Address)
				node.Synced = true
				continue
			}
			for _, object := range need {
				log.Printf("Need %s from %s\n", object.Name, server.Address)
				if object.IsDir == true {
					err := server.RequestSyncDir(object.Name, node.UUID)
					if err != nil {
						log.Error(err)
						continue
					}
				} else if object.IsDir == false {
					err := server.RequestSyncFile(object.Name, node.UUID)
					if err != nil {
						log.Error(err)
						continue
					}
				}
			}
		}
	}
	return nil
}
