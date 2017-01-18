package index

import (
	"bufio"
	"crypto/sha512"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/utils"
)

type Index struct {
	Name     string            `json:"name"`
	Checksum string            `json:"checksum,omitempty"`
	Size     int64             `json:"size"`
	ModTime  time.Time         `json:"lastModified"`
	Mode     os.FileMode       `json:"fileMode"`
	IsDir    bool              `json:"isDir"`
	Files    map[string]*Index `json:"files,omitempty"`
}

// GetChecksum returns the SHA512 hash of the file at 'path'.
func GetChecksum(path string) (string, error) {
	defer utils.TimeTrack(time.Now(), "index/GetChecksum()")
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha512.New()
	buf := bufio.NewReader(file)

	_, err = buf.WriteTo(hash)
	if err != nil {
		return "", err
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	return sum, nil
}

func NewIndex(name string, size int64, modtime time.Time, mode os.FileMode, isDir bool) *Index {
	var checksum string
	var err error
	if isDir == false {
		checksum, err = GetChecksum(name)
		if err != nil {
			return nil
		}
	} else {
		checksum = ""
	}
	return &Index{name, checksum, size, modtime, mode, isDir, nil}
}

//Recursively genearate an index for dirPath
func GetIndex(dirPath string) (map[string]*Index, error) {
	defer utils.TimeTrack(time.Now(), "index/GetIndex()")
	if dirPath == "/" || dirPath == "../" || dirPath == ".." {
		dirPath = "./"
	}
	list, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	index := make(map[string]*Index)
	for _, child := range list {
		if child.Name() == options.Config.NodeMetadataFile {
			continue
		}
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
