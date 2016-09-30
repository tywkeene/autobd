//package server provides all the necessary logic when interacting with
//an autobd server. Node side logic resides in package node
package server

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/packing"
	"github.com/tywkeene/autobd/version"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

type Server struct {
	Address     string       //Server URL
	MissedBeats int          //How many heartbeats the server has missed
	Online      bool         //Is there server online
	Client      *http.Client //connection configuration for this server
}

func NewServer(address string) *Server {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return &Server{address, 0, true, client}
}

func (server *Server) constructUrl(str string) string {
	return server.Address + "/v" + version.Major() + str
}

func DeflateResponse(resp *http.Response) ([]byte, error) {
	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return ioutil.ReadAll(reader)
}

func (server *Server) Get(endpoint string) ([]byte, error) {
	url := server.constructUrl(endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "Autobd-node/"+version.Server())
	resp, err := server.Client.Do(req)
	if err != nil {
		return nil, err
	}
	return DeflateResponse(resp)
}

func writeFile(filename string, source io.Reader) error {
	writer, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	defer writer.Close()
	io.Copy(writer, source)
	return nil
}

func (server *Server) RequestVersion() (*version.VersionInfo, error) {
	log.Println("(??) Requesting version from", server.Address)
	resp, err := http.Get(server.Address + "/version")
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

func (server *Server) RequestIndex(dir string, uuid string) (map[string]*index.Index, error) {
	log.Printf(" (??) Requesting index for directory %s from %s", dir, server.Address)
	buffer, err := server.Get("/index?dir=" + dir + "&uuid=" + uuid)
	if err != nil {
		return nil, err
	}

	remoteIndex := make(map[string]*index.Index)
	if err := json.Unmarshal(buffer, &remoteIndex); err != nil {
		return nil, err
	}
	return remoteIndex, nil
}

func (server *Server) RequestSyncDir(file string, uuid string) error {
	log.Printf(" (REQ) Requesting sync of directory '%s' from %s", file, server.Address)
	buffer, err := server.Get("/sync?grab=" + file + "&uuid=" + uuid)
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

func (server *Server) RequestSyncFile(file string, uuid string) error {
	log.Printf(" (REQ) Requesting sync of file '%s' from %s", file, server.Address)
	buffer, err := server.Get("/sync?grab=" + file + "&uuid=" + uuid)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buffer)
	err = writeFile(file, reader)
	return err
}

func compareDirs(local map[string]*index.Index, remote map[string]*index.Index) []*index.Index {
	need := make([]*index.Index, 0)
	for objName, object := range remote {
		_, existsLocally := local[object.Name] //Does it exist on the node?

		if existsLocally == false {
			need = append(need, remote[objName])
			continue
		}

		// If it does, and it's a directory, and it has children
		if existsLocally == true && object.IsDir == true && object.Files != nil {
			dirNeed := compareDirs(local[objName].Files, object.Files) //Scan the children
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
				log.Println(" (!=) Checksum mismatch:", objName)
				need = append(need, remote[objName])
				continue
			}
		}
	}
	return need
}

func (server *Server) CompareIndex(target string, uuid string) ([]*index.Index, error) {
	remoteIndex, err := server.RequestIndex(target, uuid)
	if err != nil {
		return nil, err
	}
	localIndex, err := index.GetIndex("/")
	if err != nil {
		return nil, err
	}
	need := compareDirs(localIndex, remoteIndex)
	return need, nil
}

func (server *Server) IdentifyWithServer(uuid string) ([]byte, error) {
	return server.Get("/identify?uuid=" + uuid + "&version=" + version.Server())
}

func (server *Server) SendHeartbeat(uuid string) ([]byte, error) {
	return server.Get("/heartbeat?uuid=" + uuid)
}
