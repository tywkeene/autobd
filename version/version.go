package version

import "fmt"

var (
	CommitHash string
	APIVer     string
	ServerVer  string
)

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
