package node_test

import (
	"github.com/BurntSushi/toml"
	"github.com/tywkeene/autobd/node"
	"github.com/tywkeene/autobd/options"
	"log"
	"os"
	"testing"
)

func getNodeConfig() options.NodeConf {
	var config options.NodeConf
	var configFile string = "../etc/config.toml.node"

	if _, err := toml.DecodeFile(configFile, config); err != nil {
		log.Fatal(err)
	}
	return config
}

func TestInitNode(t *testing.T) {
	config := getNodeConfig()

	if n := node.InitNode(config); n == nil {
		t.Fatal("Failed to allocate new node")
	}
}

func WriteNodeUUID(t *testing.T) {
	config := getNodeConfig()
	n := node.InitNode(config)
	if _, err := os.Stat(config.UUIDPath); os.IsNotExist(err) {
		os.Remove(config.UUIDPath)
	}
	if err := n.WriteNodeUUID(); err != nil {
		t.Fatal(err)
	}
}

func ReadNodeUUID(t *testing.T) {
	config := getNodeConfig()
	n := node.InitNode(config)
	if _, err := os.Stat(config.UUIDPath); os.IsNotExist(err) {
		n.WriteNodeUUID()
	}

	if err := n.ReadNodeUUID(); err != nil {
		t.Fatal(err)
	}
}
