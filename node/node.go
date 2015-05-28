package node

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/helpers"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
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

func RequestSync(seed string, file string) error {
	log.Printf("Requesting sync of file '%s' from %s", file, seed)
	url := seed + "/" + api.ApiVersion + "/sync?grab=" + file
	resp, err := Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") == "application/x-tar" {
		err := helpers.UnpackDir(resp.Body)
		if err != nil && err == io.EOF {
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
	err = helpers.WriteFile(file, resp.Body)
	return err
}
