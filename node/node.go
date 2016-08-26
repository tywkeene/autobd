package node

import (
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/tywkeene/autobd/manifest"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/packing"
	"github.com/tywkeene/autobd/version"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type Server struct {
	Address     string
	MissedBeats int //How many heartbeats the server has missed
	Online      bool
}

type Node struct {
	Servers []*Server
	UUID    string
	Config  options.NodeConf
}

var localNode *Node

func newServer(address string) *Server {
	return &Server{address, 0, true}
}

func newNode(config options.NodeConf) *Node {
	servers := make([]*Server, 0)
	for _, server := range config.Seeds {
		servers = append(servers, newServer(server))
	}
	UUID := uuid.NewV4().String()
	log.Println("Generated node UUID:", UUID)
	return &Node{servers, UUID, config}
}

func InitNode(config options.NodeConf) *Node {
	node := newNode(config)
	return node
}

func constructUrl(server string, str string) string {
	return server + "/v" + version.Major() + str
}

func Get(url string) (*http.Response, error) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "Autobd-node/"+version.Server())
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		buffer, _ := DeflateResponse(resp)
		var err string
		json.Unmarshal(buffer, &err)
		return nil, errors.New(err)
	}
	return resp, nil
}

func DeflateResponse(resp *http.Response) ([]byte, error) {
	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buffer []byte
	buffer, _ = ioutil.ReadAll(reader)
	return buffer, nil
}

func WriteFile(filename string, source io.Reader) error {
	writer, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	defer writer.Close()

	gr, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer gr.Close()

	io.Copy(writer, gr)
	return nil
}

func (node *Node) RequestVersion(seed string) (*version.VersionInfo, error) {
	log.Println("Requesting version from", seed)
	url := seed + "/version"
	resp, err := Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer, err := DeflateResponse(resp)
	if err != nil {
		return nil, err
	}

	var ver *version.VersionInfo
	if err := json.Unmarshal(buffer, &ver); err != nil {
		return nil, err
	}
	return ver, nil
}

func (node *Node) RequestManifest(seed string, dir string) (map[string]*manifest.Manifest, error) {
	log.Printf("Requesting manifest for directory %s from %s", dir, seed)
	endpoint := constructUrl(seed, "/manifest?dir="+dir+"&uuid="+node.UUID)
	resp, err := Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer, err := DeflateResponse(resp)
	if err != nil {
		return nil, err
	}

	remoteManifest := make(map[string]*manifest.Manifest)
	if err := json.Unmarshal(buffer, &remoteManifest); err != nil {
		return nil, err
	}
	return remoteManifest, nil
}

func (node *Node) RequestSync(seed string, file string) error {
	log.Printf("Requesting sync of file '%s' from %s", file, seed)
	endpoint := constructUrl(seed, "/sync?grab="+file+"&uuid="+node.UUID)
	resp, err := Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") == "application/x-tar" {
		err := packing.UnpackDir(resp.Body)
		if err != nil {
			return nil
		} else {
			return err
		}
	}

	//make sure we create the directory tree if it's needed
	if tree := path.Dir(file); tree != "" {
		err := os.MkdirAll(tree, 0777)
		if err != nil {
			return err
		}
	}
	err = WriteFile(file, resp.Body)
	return err
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

func compareDirs(local map[string]*manifest.Manifest, remote map[string]*manifest.Manifest) []string {
	need := make([]string, 0)
	for name, info := range remote {
		_, exists := local[name]
		if exists == true && info.IsDir == true && remote[name].Files != nil {
			dirNeed := compareDirs(local[name].Files, remote[name].Files)
			need = append(need, dirNeed...)
		}
		if _, exists := local[name]; exists == false {
			need = append(need, name)
		}
	}
	return need
}

func (node *Node) CompareManifest(server string) ([]string, error) {
	remoteManifest, err := node.RequestManifest(server, "/")
	if err != nil {
		return nil, err
	}
	localManifest, err := manifest.GetManifest("/")
	if err != nil {
		return nil, err
	}

	need := make([]string, 0)
	for remoteName, info := range remoteManifest {
		_, exists := localManifest[remoteName]
		if info.IsDir == true && exists == true {
			dirNeed := compareDirs(localManifest[remoteName].Files, remoteManifest[remoteName].Files)
			need = append(need, dirNeed...)
			continue
		}
		if exists == false {
			need = append(need, remoteName)
		}
	}
	return need, nil
}

func (node *Node) IdentifyWithServer(server string) {
	endpoint := constructUrl(server, "/identify?uuid="+node.UUID+"&version="+version.Server())
	Get(endpoint)
}

func (node *Node) sendHeartbeat(server string) (*http.Response, error) {
	endpoint := constructUrl(server, "/heartbeat?uuid="+node.UUID)
	return Get(endpoint)
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
				_, err := node.sendHeartbeat(server.Address)
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
	node.StartHeart()
	for _, server := range node.Servers {
		remoteVer, err := node.RequestVersion(server.Address)
		if err != nil {
			return err
		}
		if options.Config.NodeConfig.IgnoreVersionMismatch == false {
			if err := node.validateServerVersion(remoteVer); err != nil {
				return err
			}
		}
		node.IdentifyWithServer(server.Address)
	}
	return nil
}

func (node *Node) UpdateLoop() error {
	if err := node.ValidateAndIdentifyServers(); err != nil {
		return err
	}
	log.Printf("Running as a node. Updating every %s with %s\n",
		node.Config.UpdateInterval, node.Config.Seeds)

	updateInterval, err := time.ParseDuration(node.Config.UpdateInterval)
	if err != nil {
		return err
	}
	for {
		time.Sleep(updateInterval)
		for _, server := range node.Servers {
			if server.Online == false {
				continue
			}
			log.Printf("Updating with %s...\n", server.Address)
			need, err := node.CompareManifest(server.Address)
			if err != nil {
				log.Println(err)
				continue
			}
			if len(need) == 0 {
				log.Println("In sync with", server)
				continue
			}
			log.Printf("Need %s from %s\n", need, server.Address)
			for _, filename := range need {
				err := node.RequestSync(server.Address, filename)
				if err != nil {
					log.Println(err)
					continue
				}
			}
		}
	}
	return nil
}
