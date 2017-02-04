package version

import (
	"encoding/json"
	"fmt"
	"strings"
)

var (
	APIVersion  string
	NodeVersion string
	CommitHash  string
)

type VersionInfo struct {
	APIVersion  string `json:"api"`
	NodeVersion string `json:"node"`
	CommitHash  string `json:"commit"`
}

func Print() {
	fmt.Printf("\tAutobd (API %s) (Node %s) (git commit %s)\n", APIVersion, NodeVersion, CommitHash)
}

func Set(commit string, api string, node string) {
	CommitHash = commit
	if CommitHash == "" {
		CommitHash = "unknown"
	}
	APIVersion = api
	NodeVersion = node
}

func GetNodeVersion() string {
	return NodeVersion
}

func GetAPIVersion() string {
	return APIVersion
}

func GetCommit() string {
	return CommitHash
}

func GetMajor() string {
	return strings.Split(APIVersion, ".")[0]
}

func GetMinor() string {
	return strings.Split(APIVersion, ".")[1]
}

func GetPatch() string {
	return strings.Split(APIVersion, ".")[2]
}

func JSON() string {
	serial, _ := json.MarshalIndent(&VersionInfo{GetAPIVersion(), GetNodeVersion(), GetCommit()}, " ", " ")
	return string(serial)
}
