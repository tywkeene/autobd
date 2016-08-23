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

var UUID string

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

func RequestVersion(seed string) (*version.VersionInfo, error) {
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

func RequestManifest(seed string, dir string) (map[string]*manifest.Manifest, error) {
	log.Printf("Requesting manifest for directory %s from %s", dir, seed)
	endpoint := constructUrl(seed, "/manifest?dir="+dir+"&uuid="+UUID)
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

func RequestSync(seed string, file string) error {
	log.Printf("Requesting sync of file '%s' from %s", file, seed)
	endpoint := constructUrl(seed, "/sync?grab="+file+"&uuid="+UUID)
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

func validateServerVersion(remote *version.VersionInfo) error {
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

func CompareManifest(server string) ([]string, error) {
	remoteManifest, err := RequestManifest(server, "/")
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

func IdentifyWithServer(server string) {
	UUID = uuid.NewV4().String()
	log.Println("Generated node UUID:", UUID)
	endpoint := constructUrl(server, "/identify?uuid="+UUID+"&version="+version.Server())
	Get(endpoint)
}

func UpdateLoop(config options.NodeConf) error {
	log.Printf("Running as a node. Updating every %s with %s\n",
		config.UpdateInterval, config.Seeds)

	for _, server := range config.Seeds {
		remoteVer, err := RequestVersion(server)

		if err != nil {
			log.Println(err)
			continue
		}
		if options.Config.NodeConfig.IgnoreVersionMismatch == false {
			if err := validateServerVersion(remoteVer); err != nil {
				return err
			}
		}
		IdentifyWithServer(server)
	}

	updateInterval, err := time.ParseDuration(config.UpdateInterval)
	if err != nil {
		return err
	}

	for {
		time.Sleep(updateInterval)
		for _, server := range config.Seeds {
			log.Printf("Updating with %s...\n", server)
			need, err := CompareManifest(server)
			if err != nil {
				log.Println(err)
				continue
			}
			if len(need) == 0 {
				log.Println("In sync with", server)
				continue
			}
			log.Printf("Need %s from %s\n", need, server)
			for _, filename := range need {
				err := RequestSync(server, filename)
				if err != nil {
					log.Println(err)
					continue
				}
			}
		}
	}
	return nil
}
