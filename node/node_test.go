package node

import (
	"github.com/BurntSushi/toml"
	"github.com/tywkeene/autobd/options"
	"testing"
)

func TestInitNode(t *testing.T) {
	var config *options.NodeConf
	var configFile string = "../etc/config.toml.node"

	if _, err := toml.DecodeFile(configFile, config); err != nil {
		t.Fatal(err)
	}

	if n := InitNode(config); n == nil {
		t.Fatal("Failed to allocate new node")
	}
}
