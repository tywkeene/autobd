package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/node"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/version"
	"net/http"
	"os"
	"runtime"
)

var (
	CommitHash string
	APIVer     string
	NodeVer    string
)

func init() {
	options.GetOptions()
	if err := os.Chdir(options.Config.Root); err != nil {
		log.Panic(err)
	}
	version.Set(CommitHash, APIVer, NodeVer)
	if options.Config.VersionJSON == true {
		fmt.Println(version.JSON())
		os.Exit(0)
	}
	version.Print()
	if options.Config.Version == true {
		os.Exit(0)
	}
	if options.Config.RunNode == false {
		api.SetupRoutes()
		if err := api.ReadNodeMetadata(options.Config.NodeMetadataFile); err != nil {
			log.Warn(err)
		}
	}
}

func printLogo() {
	const node = `
 	 █████╗ ██╗   ██╗████████╗ ██████╗ ██████╗ ██████╗       ███╗   ██╗ ██████╗ ██████╗ ███████╗
	██╔══██╗██║   ██║╚══██╔══╝██╔═══██╗██╔══██╗██╔══██╗      ████╗  ██║██╔═══██╗██╔══██╗██╔════╝
	███████║██║   ██║   ██║   ██║   ██║██████╔╝██║  ██║█████╗██╔██╗ ██║██║   ██║██║  ██║█████╗
	██╔══██║██║   ██║   ██║   ██║   ██║██╔══██╗██║  ██║╚════╝██║╚██╗██║██║   ██║██║  ██║██╔══╝
	██║  ██║╚██████╔╝   ██║   ╚██████╔╝██████╔╝██████╔╝      ██║ ╚████║╚██████╔╝██████╔╝███████╗
	╚═╝  ╚═╝ ╚═════╝    ╚═╝    ╚═════╝ ╚═════╝ ╚═════╝       ╚═╝  ╚═══╝ ╚═════╝ ╚═════╝ ╚══════╝
	Backing you up since whenever...
	`
	const server = `
 	 █████╗ ██╗   ██╗████████╗ ██████╗ ██████╗ ██████╗       ███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗
	██╔══██╗██║   ██║╚══██╔══╝██╔═══██╗██╔══██╗██╔══██╗      ██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗
	███████║██║   ██║   ██║   ██║   ██║██████╔╝██║  ██║█████╗███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝
	██╔══██║██║   ██║   ██║   ██║   ██║██╔══██╗██║  ██║╚════╝╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗
	██║  ██║╚██████╔╝   ██║   ╚██████╔╝██████╔╝██████╔╝      ███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║
	╚═╝  ╚═╝ ╚═════╝    ╚═╝    ╚═════╝ ╚═════╝ ╚═════╝       ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝
	Backing you up since right now...
	`
	if options.Config.RunNode == true {
		fmt.Println(node)
	} else {
		fmt.Println(server)
	}
}

func main() {
	printLogo()
	if options.Config.RunNode == true {
		localNode := node.InitNode(options.Config.NodeConfig)
		if err := localNode.UpdateLoop(); err != nil {
			log.Panic(err)
		}
	}
	if options.Config.Cores > runtime.NumCPU() {
		log.Error("Requested processor value greater than number of actual processors, using default")
	} else {
		runtime.GOMAXPROCS(options.Config.Cores)
	}
	log.Printf("Serving '%s' on port %s", options.Config.Root, options.Config.ApiPort)
	if options.Config.Ssl == true {
		log.Infof("Using certificate (%s) and key (%s) for SSL\n", options.Config.Cert, options.Config.Key)
		log.Panic(http.ListenAndServeTLS(":"+options.Config.ApiPort, options.Config.Cert, options.Config.Key, nil))
	} else {
		log.Panic(http.ListenAndServe(":"+options.Config.ApiPort, nil))
	}
}
