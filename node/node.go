//Package node provides the node side logic of the node.
package node

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/server"
	"github.com/tywkeene/autobd/version"
	"log"
	"strings"
	"time"
)

type Node struct {
	Servers map[string]*server.Server
	UUID    string
	Config  options.NodeConf
}

var localNode *Node

func newNode(config options.NodeConf) *Node {
	servers := make(map[string]*server.Server, 0)
	for _, url := range config.Servers {
		servers[url] = server.NewServer(url)
	}
	UUID := uuid.NewV4().String()
	log.Println("Generated node UUID:", UUID)
	return &Node{servers, UUID, config}
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
				_, err := server.SendHeartbeat(node.UUID)
				if err != nil {
					log.Println(err)
					server.MissedBeats++
					if server.MissedBeats == node.Config.MaxMissedBeats {
						server.Online = false
						log.Println(server.Address, "has missed max beats, ignoring")
					}
				}
			}
		}
	}(node.Config)
}

func (node *Node) ValidateAndIdentifyServers() error {
	for _, server := range node.Servers {
		remoteVer, err := server.RequestVersion()
		if remoteVer == nil || err != nil {
			return err
		}
		if options.Config.NodeConfig.IgnoreVersionMismatch == false {
			if err := node.validateServerVersion(remoteVer); err != nil {
				log.Println(err)
				return err
			}
		}
		_, err = server.IdentifyWithServer(node.UUID)
		if err != nil {
			log.Println(err)
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
	log.Printf("Running as a node. Updating every %s with %s\n",
		node.Config.UpdateInterval, node.Config.Servers)

	updateInterval, err := time.ParseDuration(node.Config.UpdateInterval)
	if err != nil {
		return err
	}
	for {
		time.Sleep(updateInterval)
		for _, server := range node.Servers {
			if server.Online == false {
				log.Printf("(!!) (offline) Skipping %s...\n", server.Address)
				continue
			}
			log.Printf(" (??) Updating with %s...\n", server.Address)
			need, err := server.CompareIndex(node.Config.TargetDirectory, node.UUID)
			if err != nil {
				log.Println(err)
				continue
			}

			if len(need) == 0 {
				log.Println("(OK) In sync with", server.Address)
				continue
			}
			for _, object := range need {
				log.Printf(" (!=) Need %s from %s\n", object.Name, server.Address)
				if object.IsDir == true {
					err := server.RequestSyncDir(object.Name, node.UUID)
					if err != nil {
						log.Println(err)
						continue
					}
				} else if object.IsDir == false {
					err := server.RequestSyncFile(object.Name, node.UUID)
					if err != nil {
						log.Println(err)
						continue
					}
				}
			}
		}
	}
	return nil
}
