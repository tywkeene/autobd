//Package node provides the node side logic of the node.
package node

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
	"github.com/tywkeene/autobd/client"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/version"
	"strings"
	"time"
)

type Node struct {
	Servers map[string]*client.Client
	UUID    string
	Synced  bool
	Config  *options.NodeConf
}

var localNode *Node

func newNode(config *options.NodeConf) *Node {
	servers := make(map[string]*client.Client, 0)
	for _, url := range config.Servers {
		servers[url] = client.NewClient(url)
	}
	UUID := uuid.NewV4().String()
	log.Info("Generated node UUID: ", UUID)
	return &Node{servers, UUID, false, config}
}

func InitNode(config *options.NodeConf) *Node {
	node := newNode(config)
	return node
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
	go func(config *options.NodeConf) {
		interval, _ := time.ParseDuration(config.HeartbeatInterval)
		log.Info("Started heartbeat, updating every ", interval)
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

func (node *Node) SyncUp(need []*index.Index, s *client.Client) {
	for _, object := range need {
		log.Printf("Need %s from %s\n", object.Name, s.Address)
		if object.IsDir == true {
			err := s.RequestSyncDir(object.Name, node.UUID)
			if err != nil {
				log.Error(err)
				continue
			}
		} else if object.IsDir == false {
			err := s.RequestSyncFile(object.Name, node.UUID)
			if err != nil {
				log.Error(err)
				continue
			}
		}
	}
}

func (node *Node) UpdateLoop() error {
	if err := node.ValidateAndIdentifyWithServers(); err != nil {
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
		for _, s := range node.Servers {
			if s.Online == false {
				log.Info("Skipping offline server: ", s.Address)
				continue
			}
			need, err := s.CompareIndex(node.Config.TargetDirectory, node.UUID)
			if err != nil {
				log.Error(err)
				continue
			}

			if len(need) == 0 {
				node.Synced = true
				continue
			}
			node.SyncUp(need, s)
		}
	}
	return nil
}
