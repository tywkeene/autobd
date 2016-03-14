package main

import (
	"fmt"
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/node"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/version"
	"log"
	"net/http"
	"os"
	"runtime"
)

var (
	CommitHash string
	ServerVer  string
)

func init() {
	version.Set(CommitHash, ServerVer)
	version.Print()
	options.GetOptions()
	api.SetupRoutes()
}

const Logo = `
 █████╗ ██╗   ██╗████████╗ ██████╗ ██████╗ ██████╗
██╔══██╗██║   ██║╚══██╔══╝██╔═══██╗██╔══██╗██╔══██╗
███████║██║   ██║   ██║   ██║   ██║██████╔╝██║  ██║
██╔══██║██║   ██║   ██║   ██║   ██║██╔══██╗██║  ██║
██║  ██║╚██████╔╝   ██║   ╚██████╔╝██████╔╝██████╔╝
╚═╝  ╚═╝ ╚═════╝    ╚═╝    ╚═════╝ ╚═════╝ ╚═════╝
Backing you up since whenever..
`

func main() {
	if err := os.Chdir(options.Config.Root); err != nil {
		panic(err)
	}
	fmt.Println(Logo)
	if options.Config.RunNode == true {
		err := node.UpdateLoop(options.Config.NodeConfig)
		if err != nil {
			panic(err)
		}
	}
	if options.Config.Cores > runtime.NumCPU() {
		log.Println("Requested processor value greater than number of actual processors, using default")
	} else {
		log.Printf("Using %d processors\n", options.Config.Cores)
		runtime.GOMAXPROCS(options.Config.Cores)
	}
	log.Printf("Serving '%s' on port %s", options.Config.Root, options.Config.ApiPort)
	if options.Config.Ssl == true {
		log.Panic(http.ListenAndServeTLS(":"+options.Config.ApiPort, options.Config.Cert, options.Config.Key, nil))
	} else {
		log.Panic(http.ListenAndServe(":"+options.Config.ApiPort, nil))
	}
}
