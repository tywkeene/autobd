package version

import "fmt"

var (
	CommitHash string
	APIVer     string
	ServerVer  string
)

type VersionInfo struct {
	ServerVer  string `json:"server"`
	APIVer     string `json:"api"`
	CommitHash string `json:"commit"`
	Comment    string `json:"comment"`
}

func Print() {
	fmt.Printf("Autobd version %s (API v%s) (git commit %s)\n", ServerVer, APIVer, CommitHash)
}

func Set(commit string, api string, server string) {
	CommitHash = commit
	if CommitHash == "" {
		CommitHash = "unknown"
	}
	APIVer = api
	ServerVer = server
}

func Server() string {
	return ServerVer
}

func API() string {
	return APIVer
}

func Commit() string {
	return CommitHash
}
