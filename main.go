package main

import (
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/node"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/version"
	"log"
	"net/http"
	"os"
	"syscall"
)

var (
	CommitHash string
	ServerVer  string
	APIVer     string
)

func init() {
	version.Set(CommitHash, APIVer, ServerVer)
	version.Print()
	options.GetOptions()
	api.SetupRoutes()
}

func main() {
	if options.Config.RunNode == true {
		err := node.UpdateLoop(options.Config.NodeConfig)
		if err != nil {
			panic(err)
		}
	}
	if err := syscall.Chroot(options.Config.Root); err != nil {
		panic("chroot: " + err.Error())
	}
	if err := os.Chdir(options.Config.Root); err != nil {
		panic(err)
	}
	log.Printf("Serving '%s' on port %s", options.Config.Root, options.Config.ApiPort)
	log.Panic(http.ListenAndServe(":"+options.Config.ApiPort, nil))
}
