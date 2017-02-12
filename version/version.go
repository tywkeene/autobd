package version

import (
	"encoding/json"
	"fmt"
	"strings"
)

var (
	Version    string
	CommitHash string
)

type VersionInfo struct {
	Version    string `json:"version"`
	CommitHash string `json:"commit"`
}

func Print() {
	fmt.Printf("\tAutobd-%s (git commit %s)\n", Version, CommitHash)
}

func GetVersion() string {
	return Version
}

func GetCommit() string {
	return CommitHash
}

func GetMajor() string {
	return strings.Split(Version, ".")[0]
}

func GetMinor() string {
	return strings.Split(Version, ".")[1]
}

func GetPatch() string {
	return strings.Split(Version, ".")[2]
}

func JSON() string {
	serial, _ := json.MarshalIndent(&VersionInfo{GetVersion(), GetCommit()}, " ", " ")
	return string(serial)
}
