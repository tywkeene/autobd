package options

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

type NodeConf struct {
	Seed           string `toml:"seed_server"`
	UpdateInterval string `toml:"update_interval"`
}

type Conf struct {
	Root       string   `toml:"root_dir"`
	ApiPort    string   `toml:"api_port"`
	RunNode    bool     `toml:"run_as_node"`
	NodeConfig NodeConf `toml:"node"`
}

var Config Conf

func GetOptions() {
	var configFile string

	flag.StringVar(&configFile, "config", "", "Configuration file")
	flag.StringVar(&Config.Root, "root", "", "Root directory to serve (required)")
	flag.StringVar(&Config.ApiPort, "api-port", "8081", "Port that the API listens on")
	flag.BoolVar(&Config.RunNode, "node", false, "Run as a node")
	flag.StringVar(&Config.NodeConfig.Seed, "seed", "", "Seed server to connect with (required when running as a node)")
	flag.StringVar(&Config.NodeConfig.UpdateInterval, "update-interval", "1m", "How often to update with the other servers")

	flag.Parse()

	if configFile != "" {
		if _, err := toml.DecodeFile(configFile, &Config); err != nil {
			fmt.Printf("Error reading config %s: %s\n", configFile, err.Error())
			os.Exit(-1)
		}
		fmt.Printf("Configration file options in %s overriding command line options\n", configFile)
	}

	if Config.Root == "" {
		fmt.Println("Must specify root directory")
		os.Exit(-1)
	}

	if Config.RunNode == true && Config.NodeConfig.Seed == "" {
		fmt.Println("Must specify seed server when running as node")
		os.Exit(-1)
	}
}
