package index

import (
	"bufio"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"
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

func GetChecksum(path string) (string, error) {
	file, err := os.Open(path)

	if err != nil {
		return "", err
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return "", err
	}

	size := stats.Size()
	raw := make([]byte, size)

	buffer := bufio.NewReader(file)
	_, err = buffer.Read(raw)

	hash := sha512.New()
	io.WriteString(hash, string(raw))
	checksum := hex.EncodeToString(hash.Sum(nil))

	return checksum, nil
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
