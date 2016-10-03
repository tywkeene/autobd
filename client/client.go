//package client provides all the necessary logic when interacting with
//an autobd server. Node side logic resides in package node
package client

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/packing"
	"github.com/tywkeene/autobd/version"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
)

//The Client struct describes a connection to a server, it's status, and an http client
type Client struct {
	Address     string       //Server URL
	MissedBeats int          //How many heartbeats the server has missed
	Online      bool         //Is this server online
	http        *http.Client //connection configuration for this server
}

func NewClient(address string) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return &Client{address, 0, true, client}
}

func (client *Client) constructUrl(str string) string {
	return client.Address + "/v" + version.GetMajor() + str
}

func DeflateResponse(resp *http.Response) ([]byte, error) {
	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return ioutil.ReadAll(reader)
}

func (client *Client) Get(endpoint string) ([]byte, error) {
	url := client.constructUrl(endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "Autobd-node/"+version.GetAPIVersion())
	resp, err := client.http.Do(req)
	if err != nil {
		return nil, err
	}
	return DeflateResponse(resp)
}

func writeFile(filename string, source io.Reader) error {
	writer, err := os.Create(filename)
	if err != nil {
		log.Error(err)
		return err
	}
	defer writer.Close()
	io.Copy(writer, source)
	return nil
}

func (client *Client) RequestVersion() (*version.VersionInfo, error) {
	resp, err := http.Get(client.Address + "/version")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ver *version.VersionInfo
	buffer, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(buffer, &ver); err != nil {
		return nil, err
	}
	return ver, nil
}

func (client *Client) RequestIndex(dir string, uuid string) (map[string]*index.Index, error) {
	buffer, err := client.Get("/index?dir=" + dir + "&uuid=" + uuid)
	if err != nil {
		return nil, err
	}

	remoteIndex := make(map[string]*index.Index)
	if err := json.Unmarshal(buffer, &remoteIndex); err != nil {
		return nil, err
	}
	return remoteIndex, nil
}

func (client *Client) RequestSyncDir(file string, uuid string) error {
	buffer, err := client.Get("/sync?grab=" + file + "&uuid=" + uuid)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(buffer)
	if err := packing.UnpackDir(reader); err != nil {
		return err
	}

	//make sure we create the directory tree if it's needed
	if tree := path.Dir(file); tree != "" {
		err := os.MkdirAll(tree, 0777)
		if err != nil {
			return err
		}
	}
	err = writeFile(file, reader)
	return err
}

func (client *Client) RequestSyncFile(file string, uuid string) error {
	buffer, err := client.Get("/sync?grab=" + file + "&uuid=" + uuid)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buffer)
	err = writeFile(file, reader)
	return err
}

func CompareDirs(local map[string]*index.Index, remote map[string]*index.Index) []*index.Index {
	need := make([]*index.Index, 0)
	for objName, object := range remote {
		_, existsLocally := local[object.Name] //Does it exist on the node?

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

func (client *Client) CompareIndex(target string, uuid string) ([]*index.Index, error) {
	remoteIndex, err := client.RequestIndex(target, uuid)
	if err != nil {
		return nil, err
	}
	localIndex, err := index.GetIndex("/")
	if err != nil {
		return nil, err
	}
	need := CompareDirs(localIndex, remoteIndex)
	return need, nil
}

func (client *Client) IdentifyWithServer(uuid string) ([]byte, error) {
	return client.Get("/identify?uuid=" + uuid + "&version=" + version.GetAPIVersion())
}

func (client *Client) SendHeartbeat(uuid string, synced bool) ([]byte, error) {
	return client.Get("/heartbeat?uuid=" + uuid + "&synced=" + strconv.FormatBool(synced))
}
