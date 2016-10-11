//package client provides all the necessary logic when interacting with
//an autobd server. Node side logic resides in package node
package client

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/packing"
	"github.com/tywkeene/autobd/utils"
	"github.com/tywkeene/autobd/version"
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
func (client *Client) Get(endpoint string, queryValues map[string]string, userAgent string) ([]byte, error) {
	request := client.ConstructRequest(endpoint, queryValues)
	request.Header.Set("Accept-Encoding", "application/x-gzip")
	request.Header.Set("User-Agent", userAgent)
	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	return InflateResponse(response)
}

func (client *Client) RequestVersion() ([]byte, error) {
	resp, err := http.Get(client.Address + "/version")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (client *Client) RequestIndex(dir string, uuid string, userAgent string) ([]byte, error) {
	queryValues := make(map[string]string)
	queryValues["dir"] = dir
	queryValues["uuid"] = uuid
	return client.Get("/index", queryValues, userAgent)
}

func (client *Client) RequestSyncDir(dir string, uuid string, userAgent string) error {
	queryValues := make(map[string]string)
	queryValues["grab"] = dir
	queryValues["uuid"] = uuid
	buffer, err := client.Get("/sync", queryValues, userAgent)
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
	return utils.WriteFile(dir, reader)
}

func (client *Client) RequestSyncFile(file string, uuid string, userAgent string) error {
	queryValues := make(map[string]string)
	queryValues["grab"] = file
	queryValues["uuid"] = uuid
	buffer, err := client.Get("/sync", queryValues, userAgent)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buffer)
	return utils.WriteFile(file, reader)
}

//Identify with a server and tell it the node's version and uuid
func (client *Client) IdentifyWithServer(uuid string, userAgent string) ([]byte, error) {
	queryValues := make(map[string]string)
	queryValues["uuid"] = uuid
	queryValues["version"] = version.GetNodeVersion()
	return client.Get("/identify", queryValues, userAgent)
}

//Send a heartbeat to a server, updating the node's synced status
func (client *Client) SendHeartbeat(uuid string, synced bool, userAgent string) ([]byte, error) {
	queryValues := make(map[string]string)
	queryValues["uuid"] = uuid
	queryValues["synced"] = strconv.FormatBool(synced)
	return client.Get("/heartbeat", queryValues, userAgent)
}

func (client *Client) GetNodes(uuid string, userAgent string) ([]byte, error) {
	queryValues := make(map[string]string)
	queryValues["uuid"] = uuid
	return client.Get("/nodes", queryValues, userAgent)
}
