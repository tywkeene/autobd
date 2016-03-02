package manifest

import (
	"io/ioutil"
	"os"
	"path"
	"time"
)

type Manifest struct {
	Name    string               `json:"name"`
	Size    int64                `json:"size"`
	ModTime time.Time            `json:"lastModified"`
	Mode    os.FileMode          `json:"fileMode"`
	IsDir   bool                 `json:"isDir"`
	Files   map[string]*Manifest `json:"files,omitempty"`
}

func NewManifest(name string, size int64, modtime time.Time, mode os.FileMode, isDir bool) *Manifest {
	return &Manifest{name, size, modtime, mode, isDir, nil}
}

func GetManifest(dirPath string) (map[string]*Manifest, error) {
	if dirPath == "/" || dirPath == "../" || dirPath == ".." {
		dirPath = "./"
	}
	list, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	manifest := make(map[string]*Manifest)
	for _, child := range list {
		childPath := path.Join(dirPath, child.Name())
		manifest[childPath] = NewManifest(childPath, child.Size(), child.ModTime(), child.Mode(), child.IsDir())
		if child.IsDir() == true {
			childContent, err := GetManifest(childPath)
			if err != nil {
				return nil, err
			}
			manifest[childPath].Files = childContent
		}
	}
	return manifest, nil
}
