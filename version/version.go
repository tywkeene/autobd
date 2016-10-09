package version

import (
	"encoding/json"
	"fmt"
	"strings"
)

var (
	APIVersion  string
	NodeVersion string
	CliVersion  string
	CommitHash  string
)

type VersionInfo struct {
	APIVersion  string `json:"api"`
	NodeVersion string `json:"node"`
	CliVersion  string `json:"cli"`
	CommitHash  string `json:"commit"`
}

func Print() {
	fmt.Printf("\tAutobd (API %s) (Node %s) (Cli %s) (git commit %s)\n", APIVersion, NodeVersion, CliVersion, CommitHash)
}

func Set(commit string, api string, node string, cli string) {
	CommitHash = commit
	if CommitHash == "" {
		CommitHash = "unknown"
	}
	APIVersion = api
	NodeVersion = node
	CliVersion = cli
}

func GetNodeVersion() string {
	return NodeVersion
}

func GetCliVersion() string {
	return CliVersion
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
	serial, _ := json.MarshalIndent(&VersionInfo{GetAPIVersion(), GetNodeVersion(), GetCliVersion(), GetCommit()}, " ", " ")
	return string(serial)
}
