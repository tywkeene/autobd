package options

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

type NodeConf struct {
	Servers               []string `toml:"servers"`
	UpdateInterval        string   `toml:"update_interval"`
	HeartbeatInterval     string   `toml:"heartbeat_interval"`
	MaxMissedBeats        int      `toml:"max_missed_beats"`
	IgnoreVersionMismatch bool     `toml:"node_ignore_version_mismatch"`
	TargetDirectory       string   `toml:"target_directory"`
}

type Conf struct {
	Root       string   `toml:"root_dir"`
	ApiPort    string   `toml:"api_port"`
	RunNode    bool     `toml:"run_as_node"`
	NodeConfig NodeConf `toml:"node"`
	Cores      int      `toml:"cores"`
	Server     string   `toml:"server"`
	Cert       string   `toml:"tls_cert"`
	Key        string   `toml:"tls_key"`
	Ssl        bool     `toml:"use_ssl"`
}

var Config Conf

func GetOptions() {
	var configFile string

	flag.StringVar(&configFile, "config", "", "Configuration file")
	flag.IntVar(&Config.Cores, "cores", 2, "Amount of cores to pass to GOMAXPROC (experimental)")
	flag.StringVar(&Config.Root, "root", "", "Root directory to serve (required). Must be absolute path")
	flag.StringVar(&Config.ApiPort, "api-port", "8081", "Port that the API listens on")
	flag.StringVar(&Config.Server, "server", "", "Server to query")
	flag.BoolVar(&Config.RunNode, "node", false, "Run as a node")
	flag.StringVar(&Config.Cert, "tls-cert", "", "Path to TLS certificate to use")
	flag.StringVar(&Config.Key, "tls-key", "", "Path to TLS key to use")
	flag.BoolVar(&Config.Ssl, "ssl", true, "Use TLS/SSL")

	flag.IntVar(&Config.NodeConfig.MaxMissedBeats, "missed-beats", 4, "How many heartbeats the server can miss before the node goes offline")
	flag.StringVar(&Config.NodeConfig.HeartbeatInterval, "heartbeat-interval", "30s", "How often to send a heartbeat to the server")
	flag.StringVar(&Config.NodeConfig.UpdateInterval, "update-interval", "1m", "How often to update with the other servers")
	flag.BoolVar(&Config.NodeConfig.IgnoreVersionMismatch, "node-ignore-version-mismatch", false,
		"Ignore a mismatch in server and client versions")
	flag.StringVar(&Config.NodeConfig.TargetDirectory, "target-directory", "/", "Which directory on the node to sync")

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

	if Config.RunNode == true && Config.NodeConfig.Servers == nil && Config.Server == "" {
		panic("Must specify seed server when running as node")
	} else if Config.RunNode == true && Config.NodeConfig.Servers == nil && Config.Server != "" {
		Config.NodeConfig.Servers = append(Config.NodeConfig.Servers, Config.Server)
	}
}
