package version

import (
	"fmt"
	"strings"
)

var (
	CommitHash string
	ServerVer  string
)

type VersionInfo struct {
	ServerVer  string `json:"server"`
	CommitHash string `json:"commit"`
}

func Print() {
	fmt.Printf("Autobd version %s (git commit %s)\n", ServerVer, CommitHash)
}

func Set(commit string, server string) {
	CommitHash = commit
	if CommitHash == "" {
		CommitHash = "unknown"
	}
	ServerVer = server
}

func Server() string {
	return ServerVer
}

func Commit() string {
	return CommitHash
}

func Major() string {
	return strings.Split(ServerVer, ".")[0]
}

func Minor() string {
	return strings.Split(ServerVer, ".")[1]
}

func Patch() string {
	return strings.Split(ServerVer, ".")[2]
}
