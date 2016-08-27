package server

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"github.com/tywkeene/autobd/manifest"
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
	Address     string
	MissedBeats int //How many heartbeats the server has missed
	Online      bool
	Client      *http.Client
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

	var buffer []byte
	buffer, _ = ioutil.ReadAll(reader)
	return buffer, nil
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

	gr, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer gr.Close()

	io.Copy(writer, gr)
	return nil
}

func (server *Server) RequestVersion() (*version.VersionInfo, error) {
	log.Println("Requesting version from", server.Address)
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

func (server *Server) RequestManifest(dir string, uuid string) (map[string]*manifest.Manifest, error) {
	log.Printf("Requesting manifest for directory %s from %s", dir, server.Address)
	buffer, err := server.Get("/manifest?dir=" + dir + "&uuid=" + uuid)
	if err != nil {
		return nil, err
	}

	remoteManifest := make(map[string]*manifest.Manifest)
	if err := json.Unmarshal(buffer, &remoteManifest); err != nil {
		return nil, err
	}
	return remoteManifest, nil
}

func (server *Server) RequestSync(file string, uuid string) error {
	log.Printf("Requesting sync of file '%s' from %s", file, server.Address)
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

func (server *Server) CompareManifest(uuid string) ([]string, error) {
	remoteManifest, err := server.RequestManifest("/", uuid)
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

func (server *Server) IdentifyWithServer(uuid string) ([]byte, error) {
	return server.Get("/identify?uuid=" + uuid + "&version=" + version.Server())
}

func (server *Server) SendHeartbeat(uuid string) ([]byte, error) {
	return server.Get("/heartbeat?uuid=" + uuid)
}
