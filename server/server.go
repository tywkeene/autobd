package server

import (
	log "github.com/Sirupsen/logrus"
	"github.com/tywkeene/autobd/cache"
	"github.com/tywkeene/autobd/nodelist"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/routes"
	"github.com/tywkeene/autobd/utils"
	"net/http"
)

func Launch() {
	if err := nodelist.ReadNodeList(options.Config.NodeListFile); err != nil {
		utils.HandleError(err, utils.ErrorActionWarn)
		nodelist.InitializeNodeList()
	}
	err := cache.Initialize("./")
	utils.HandlePanic(err)

	routes.SetupRoutes()
	go routes.StartHeartBeatTracker()

	log.Printf("Serving '%s' on port %s", options.Config.Root, options.Config.ApiPort)
	if options.Config.Ssl == true {
		log.Infof("Using certificate (%s) and key (%s) for SSL\n", options.Config.Cert, options.Config.Key)
		log.Panic(http.ListenAndServeTLS(":"+options.Config.ApiPort, options.Config.Cert, options.Config.Key, nil))
	} else {
		log.Panic(http.ListenAndServe(":"+options.Config.ApiPort, nil))
	}
}
