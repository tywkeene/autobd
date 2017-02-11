//package connection provides all the necessary logic when interacting with
//an autobd server. Node side logic resides in package node
package connection

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/nodelist"
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

//The Connection struct describes a connection to a server, it's status, and an http client
type Connection struct {
	Address     string       //Server URL
	MissedBeats int          //How many heartbeats the server has missed
	Online      bool         //Is this server online
	Synced      bool         //Is the node synced with this server?
	UserAgent   string       //The useragent the node will send to this server
	client      *http.Client //connection configuration for this server
}

func (connection *Connection) HandleAPIError(response *http.Response) error {
	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		buffer, err := InflateResponse(response)
		if err != nil {
			return err
		}
		var errData *utils.APIError
		if err = json.Unmarshal(buffer, &errData); err != nil {
			return err
		}
		return fmt.Errorf("Received error from [%s]->(HTTP Status:%d):%s",
			connection.Address, errData.HTTPStatus, errData.ErrorMessage)
	}
	return nil
}

func NewConnection(address string, userAgent string) *Connection {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	connection := &http.Client{Transport: tr}
	return &Connection{
		Address:     address,
		MissedBeats: 0,
		Online:      true,
		Synced:      false,
		UserAgent:   userAgent,
		client:      connection,
	}
}

func (connection *Connection) SetSynced(value bool) {
	if value != connection.Synced {
		connection.Synced = value
		switch connection.Synced {
		case true:
			log.Infof("Synced with %s", connection.Address)
			break
		case false:
			log.Infof("Out of sync with %s", connection.Address)
			break
		}
	}
}

func (connection *Connection) SetOnline(value bool) {
	if value != connection.Online {
		connection.Online = value
		switch connection.Online {
		case true:
			log.Infof("Server has come online: %s", connection.Address)
			break
		case false:
			log.Infof("Server has gone offline: %s", connection.Address)
			break
		}
	}
}

func (connection *Connection) ConstructUrl(endpoint string) string {
	urlStr, err := url.Parse(connection.Address + "/v" + version.GetMajor() + endpoint)
	if utils.HandleError(err, utils.ErrorActionErr) == true {
		return ""
	}
	return urlStr.String()
}

func (connection *Connection) SetRequestHeaders(request *http.Request) {
	request.Header.Set("Accept-Encoding", "application/x-gzip")
	request.Header.Set("User-Agent", connection.UserAgent)
}

func (connection *Connection) ConstructGetRequest(endpoint string, values map[string]string) *http.Request {
	request, err := http.NewRequest("GET", connection.ConstructUrl(endpoint), nil)
	if utils.HandleError(err, utils.ErrorActionErr) == true {
		return nil
	}
	connection.SetRequestHeaders(request)
	query := request.URL.Query()
	for name, value := range values {
		query.Add(name, value)
	}
	request.URL.RawQuery = query.Encode()
	return request
}

func (connection *Connection) ConstructPostRequest(endpoint string, data interface{}) *http.Request {
	serial, err := json.Marshal(&data)
	if utils.HandleError(err, utils.ErrorActionErr) == true {
		return nil
	}
	request, err := http.NewRequest("POST", connection.ConstructUrl(endpoint), bytes.NewBuffer(serial))
	if utils.HandleError(err, utils.ErrorActionErr) == true {
		return nil
	}
	connection.SetRequestHeaders(request)
	request.Header.Set("Content-Type", "application/json")
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
func (connection *Connection) Get(endpoint string, queryValues map[string]string) ([]byte, error) {
	request := connection.ConstructGetRequest(endpoint, queryValues)
	response, err := connection.client.Do(request)
	if err != nil {
		return nil, err
	}
	if err := connection.HandleAPIError(response); err != nil {
		return nil, err
	}
	return InflateResponse(response)
}

func (connection *Connection) Post(endpoint string, data interface{}) ([]byte, error) {
	request := connection.ConstructPostRequest(endpoint, data)
	response, err := connection.client.Do(request)
	if err != nil {
		return nil, err
	}
	if err := connection.HandleAPIError(response); err != nil {
		return nil, err
	}
	return InflateResponse(response)
}

func (connection *Connection) RequestVersion() ([]byte, error) {
	resp, err := connection.client.Get(connection.Address + "/version")
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}

func (connection *Connection) RequestIndex(dir string, uuid string) ([]byte, error) {
	queryValues := make(map[string]string)
	queryValues["dir"] = dir
	queryValues["uuid"] = uuid
	response, err := connection.Get("/index", queryValues)
	if err != nil {
		panic(err)
	}
	return response, err
}

func (connection *Connection) RequestSyncDir(dir string, uuid string) error {
	queryValues := make(map[string]string)
	queryValues["grab"] = dir
	queryValues["uuid"] = uuid
	buffer, err := connection.Get("/sync", queryValues)
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

func (connection *Connection) RequestSyncFile(file string, uuid string) error {
	queryValues := make(map[string]string)
	queryValues["grab"] = file
	queryValues["uuid"] = uuid
	buffer, err := connection.Get("/sync", queryValues)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buffer)
	return utils.WriteFile(file, reader)
}

//Identify with a server and tell it the node's version and uuid
func (connection *Connection) IdentifyWithServer(version string, uuid string) ([]byte, error) {
	metaData := &nodelist.NodeMetadata{
		Version: version,
		UUID:    uuid,
	}
	return connection.Post("/identify", &metaData)
}

//Send a heartbeat to a server, updating the node's synced status
func (connection *Connection) SendHeartbeat(uuid string) ([]byte, error) {
	heartbeat := &nodelist.NodeHeartbeat{
		UUID:   uuid,
		Synced: strconv.FormatBool(connection.Synced),
	}
	return connection.Post("/heartbeat", &heartbeat)
}

func (connection *Connection) GetNodes(uuid string) ([]byte, error) {
	queryValues := make(map[string]string)
	queryValues["uuid"] = uuid
	return connection.Get("/nodes", queryValues)
}
