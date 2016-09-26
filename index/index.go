package index

import (
	"io/ioutil"
	"os"
	"path"
	"time"
)

type Index struct {
	Name    string            `json:"name"`
	Size    int64             `json:"size"`
	ModTime time.Time         `json:"lastModified"`
	Mode    os.FileMode       `json:"fileMode"`
	IsDir   bool              `json:"isDir"`
	Files   map[string]*Index `json:"files,omitempty"`
}

func NewIndex(name string, size int64, modtime time.Time, mode os.FileMode, isDir bool) *Index {
	return &Index{name, size, modtime, mode, isDir, nil}
}

func GetIndex(dirPath string) (map[string]*Index, error) {
	if dirPath == "/" || dirPath == "../" || dirPath == ".." {
		dirPath = "./"
	}
	list, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	index := make(map[string]*Index)
	for _, child := range list {
		childPath := path.Join(dirPath, child.Name())
		index[childPath] = NewIndex(childPath, child.Size(), child.ModTime(), child.Mode(), child.IsDir())
		if child.IsDir() == true {
			childContent, err := GetIndex(childPath)
			if err != nil {
				return nil, err
			}
			index[childPath].Files = childContent
		}
	}
	return index, nil
}
