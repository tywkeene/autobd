package main

import (
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/options"
	"log"
	"net/http"
	"os"
	"syscall"
)

//Populated by linker magic
var commit string

func init() {
	api.PrintVersionInfo(commit)
	options.GetOptions()
	api.SetupRoutes()
}

func main() {
	if err := syscall.Chroot(options.Config.Root); err != nil {
		panic("chroot: " + err.Error())
	}
	if err := os.Chdir(options.Config.Root); err != nil {
		panic(err)
	}
	log.Printf("Serving '%s' on port %s", options.Config.Root, options.Config.ApiPort)
	log.Panic(http.ListenAndServe(":"+options.Config.ApiPort, nil))
}
