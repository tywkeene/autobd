package cache

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/index"
)

var rootCache map[string]*index.Index

func Initialize(rootPath string) error {
	var validPath string
	var err error
	validPath, err = index.ValidateDirectory(rootPath)
	if err != nil {
		return err
	}
	log.Infof("Generating root cache index for (%s). This may take a minute...", rootPath)
	rootCache, err = index.GetIndex(validPath)
	if err != nil {
		return err
	}
	return nil
}

func FindDirectory(dirPath string, within map[string]*index.Index) map[string]*index.Index {
	for _, item := range within {
		if item.IsDir == true {
			if item.Name == dirPath {
				return item.Files
			}
			if ret := FindDirectory(dirPath, item.Files); ret != nil {
				return ret
			}
		}
	}
	return nil
}

func Get(dirPath string) (map[string]*index.Index, error) {
	validPath, err := index.ValidateDirectory(dirPath)
	if err != nil {
		return nil, err
	}
	if validPath == "./" {
		return rootCache, nil
	}
	if ret := FindDirectory(validPath, rootCache); ret != nil {
		return ret, nil
	}
	return nil, fmt.Errorf("Could not find directory '%s'", validPath)
}
