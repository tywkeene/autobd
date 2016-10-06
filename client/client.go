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
	"net/url"
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

func (client *Client) ConstructUrl(endpoint string) string {
	urlStr, err := url.Parse(client.Address + "/v" + version.GetMajor() + endpoint)
	if err != nil {
		log.Fatal(err)
	}
	return urlStr.String()
}

func (client *Client) ConstructRequest(endpoint string, values map[string]string) *http.Request {
	request, err := http.NewRequest("GET", client.ConstructUrl(endpoint), nil)
	if err != nil {
		log.Error(err)
		return nil
	}

	query := request.URL.Query()
	for name, value := range values {
		query.Add(name, value)
	}
	request.URL.RawQuery = query.Encode()
	return request
}

//Check to see if the reponse is gzip'd, if it is, inflate it, if it's not, just return the
//normal response body as-is
func InflateResponse(resp *http.Response) ([]byte, error) {
	if resp.Header.Get("Content-Encoding") == "application/x-gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		buffer, err := ioutil.ReadAll(reader)
		return buffer, err
	}
	return ioutil.ReadAll(resp.Body)
}

//HTTP GET with autobd specific headers set, returns a gzip reader if the response is
//gzipped, a normal response body otherwise
func (client *Client) Get(endpoint string, queryValues map[string]string) ([]byte, error) {
	request := client.ConstructRequest(endpoint, queryValues)
	request.Header.Set("Accept-Encoding", "application/x-gzip")
	request.Header.Set("User-Agent", "Autobd-node/"+version.GetAPIVersion())
	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	return InflateResponse(response)
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
	queryValues := make(map[string]string)
	queryValues["dir"] = dir
	queryValues["uuid"] = uuid

	buffer, err := client.Get("/index", queryValues)
	if err != nil {
		return nil, err
	}

	remoteIndex := make(map[string]*index.Index)
	if err := json.Unmarshal(buffer, &remoteIndex); err != nil {
		return nil, err
	}
	return remoteIndex, nil
}

func (client *Client) RequestSyncDir(dir string, uuid string) error {
	queryValues := make(map[string]string)
	queryValues["grab"] = dir
	queryValues["uuid"] = uuid
	buffer, err := client.Get("/sync", queryValues)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(buffer)
	if err := packing.UnpackDir(reader); err != nil {
		return err
	}

	//make sure we create the directory tree if it's needed
	if tree := path.Dir(dir); tree != "" {
		err := os.MkdirAll(tree, 0777)
		if err != nil {
			return err
		}
	}
	return writeFile(dir, reader)
}

func (client *Client) RequestSyncFile(file string, uuid string) error {
	queryValues := make(map[string]string)
	queryValues["grab"] = file
	queryValues["uuid"] = uuid
	buffer, err := client.Get("/sync", queryValues)
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

//Identify with a server and tell it the node's version and uuid
func (client *Client) IdentifyWithServer(uuid string) ([]byte, error) {
	queryValues := make(map[string]string)
	queryValues["uuid"] = uuid
	queryValues["version"] = version.GetNodeVersion()
	return client.Get("/identify", queryValues)
}

//Send a heartbeat to a server, updating the node's synced status
func (client *Client) SendHeartbeat(uuid string, synced bool) ([]byte, error) {
	queryValues := make(map[string]string)
	queryValues["uuid"] = uuid
	queryValues["synced"] = strconv.FormatBool(synced)
	return client.Get("/heartbeat", queryValues)
}
