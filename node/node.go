package node

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"github.com/SaviorPhoenix/autobd/api"
	"io/ioutil"
	"log"
	"net/http"
)

func Get(url string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "Autobd-node/"+api.Version)
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

func RequestVersion(seed string) (*api.VersionInfo, error) {
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

	var ver *api.VersionInfo
	if err := json.Unmarshal(buffer, &ver); err != nil {
		return nil, err
	}
	return ver, nil
}

func RequestManifest(seed string, dir string) (map[string]*api.Manifest, error) {
	log.Printf("%s Requesting manifest for directory %s from %s", dir, seed)
	url := seed + "/" + api.ApiVersion + "/manifest?dir=" + dir
	resp, err := Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer, err := DeflateResponse(resp)
	if err != nil {
		return nil, err
	}

	manifest := make(map[string]*api.Manifest)
	if err := json.Unmarshal(buffer, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func RequestSync(seed string, file string) ([]byte, error) {
	log.Printf("Requesting sync of file '%s' from %s", file, seed)
	url := seed + "/" + api.ApiVersion + "/sync?grab=" + file
	resp, err := Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buffer, err := DeflateResponse(resp)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}
