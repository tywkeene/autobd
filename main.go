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
	ServerVer  string
)

func init() {
	version.Set(CommitHash, ServerVer)
	version.Print()
	options.GetOptions()
	if options.Config.RunNode == false {
		api.SetupRoutes()
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
	`
	const server = `
 █████╗ ██╗   ██╗████████╗ ██████╗ ██████╗ ██████╗       ███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗
██╔══██╗██║   ██║╚══██╔══╝██╔═══██╗██╔══██╗██╔══██╗      ██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗
███████║██║   ██║   ██║   ██║   ██║██████╔╝██║  ██║█████╗███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝
██╔══██║██║   ██║   ██║   ██║   ██║██╔══██╗██║  ██║╚════╝╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗
██║  ██║╚██████╔╝   ██║   ╚██████╔╝██████╔╝██████╔╝      ███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║
╚═╝  ╚═╝ ╚═════╝    ╚═╝    ╚═════╝ ╚═════╝ ╚═════╝       ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝
	`
	if options.Config.RunNode == true {
		fmt.Println(node)
	} else {
		fmt.Println(server)
	}
}

func main() {
	if err := os.Chdir(options.Config.Root); err != nil {
		panic(err)
	}
	printLogo()
	if options.Config.RunNode == true {
		localNode := node.InitNode(options.Config.NodeConfig)
		if err := localNode.UpdateLoop(); err != nil {
			panic(err)
		}
	}
	if options.Config.Cores > runtime.NumCPU() {
		log.Error("Requested processor value greater than number of actual processors, using default")
	} else {
		runtime.GOMAXPROCS(options.Config.Cores)
	}
	log.Printf("Serving '%s' on port %s", options.Config.Root, options.Config.ApiPort)
	if options.Config.Ssl == true {
		log.Info("Using certificate (%s) and key (%s) for SSL\n", options.Config.Cert, options.Config.Key)
		log.Panic(http.ListenAndServeTLS(":"+options.Config.ApiPort, options.Config.Cert, options.Config.Key, nil))
	} else {
		log.Panic(http.ListenAndServe(":"+options.Config.ApiPort, nil))
	}
}
