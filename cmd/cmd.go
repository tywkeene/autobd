package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/abiosoft/ishell"
	"github.com/satori/go.uuid"
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/client"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/version"
	"strings"
	"time"
)

type CliConfig struct {
	HeartbeatInterval string `toml:"heartbeat_interval"` // How often to send a heartbeat to the servers
	MaxMissedBeats    int    `toml:"max_missed_beats"`   //How many heartbeats a server is allowed to miss
	OutputJSON        bool   `toml:"output_json"`        //Output json instead of pretty printing
	SyncDirPath       string `toml:"sync_dir"`           //Where to save synced files/directories
	UUIDPath          string `toml:"uuid_path"`          //Where to read/write cli UUID
}

type Cli struct {
	Servers       []*client.Client //Clients for servers
	CurrentServer *client.Client   //Current working server
	Config        CliConfig        //Cli config
	UUID          string           //Cli UUID
}

var cliUserAgent string = "Autobd-cli/" + version.GetCliVersion()

func newCli(configFile string) *Cli {
	var config CliConfig
	UUID := uuid.NewV4().String()
	fmt.Println("cli-UUID: ", UUID)
	if configFile != "" {
		if _, err := toml.DecodeFile(configFile, &config); err != nil {
			fmt.Printf("Error reading config %s: %s\n", configFile, err.Error())
			return nil
		}
	}
	return &Cli{make([]*client.Client, 0), nil, config, UUID}
}

func (c *Cli) appendServer(server *client.Client) {
	if c.Servers == nil {
		c.Servers = make([]*client.Client, 0)
	}
	c.Servers = append(c.Servers, server)
}

func (c *Cli) addServer(address string) error {
	server := client.NewClient(address)
	_, err := server.IdentifyWithServer(c.UUID, cliUserAgent)
	if err != nil {
		return err
	}
	c.appendServer(server)
	c.CurrentServer = server
	return nil
}

func (c *Cli) findServer(address string) *client.Client {
	for _, server := range c.Servers {
		if server.Address == address {
			return server
		}
	}
	return nil
}

func (c *Cli) StartHeartbeat() {
	go func(c *Cli) {
		interval, _ := time.ParseDuration(c.Config.HeartbeatInterval)
		for {
			time.Sleep(interval)

			for _, server := range c.Servers {
				_, err := server.SendHeartbeat(c.UUID, true, cliUserAgent)
				if err != nil {
					server.MissedBeats++
					if server.MissedBeats == c.Config.MaxMissedBeats {
						server.Online = false
					}
				}
			}
		}
	}(c)
}

func truncateString(str string, length int) string {
	sub := str[:length]
	return sub
}

func (c *Cli) prettyPrintDir(dir *index.Index) {
	fmt.Printf("Name\tModify time\tMode\n")
	fmt.Println(dir.Name, dir.ModTime, dir.Mode)
}

func (c *Cli) prettyPrintFile(file *index.Index) {
	fmt.Printf("Name: %s\tChecksum: %s\tSize: %d\tModify time: %v\tMode: %v\n",
		file.Name, truncateString(file.Checksum, 8), file.Size, file.ModTime, file.Mode)
}

func (c *Cli) prettyPrintIndex(index map[string]*index.Index) {
	for _, object := range index {
		if object.IsDir == true {
			c.prettyPrintIndex(object.Files)
			c.prettyPrintDir(object)
			continue
		} else {
			c.prettyPrintFile(object)
		}
	}
}

func (c *Cli) prettyPrintNodes(nodes map[string]*api.Node) {
	for uuid, node := range nodes {
		fmt.Println("-----------------------------")
		if uuid == c.UUID {
			fmt.Print("* ")
		}
		fmt.Printf("UUID: %s\nAddress: %s\nVersion: %s\nLast online: %s\nCurrently online: %v\nSynced: %v\n",
			uuid, node.Address, node.Version, node.LastOnline, node.IsOnline, node.Synced)
	}
	fmt.Println("-----------------------------")
}

func (c *Cli) printConfig() {
	fmt.Printf("heartbeat_interval = %s\nmax_missed_beats = %d\noutput_json = %v\n",
		c.Config.HeartbeatInterval, c.Config.MaxMissedBeats, c.Config.OutputJSON)
}

func Start() {
	shell := ishell.New()

	c := newCli(options.Config.CliConfigPath)
	c.printConfig()

	shell.Register("print-config", func(args ...string) (string, error) {
		c.printConfig()
		return "OK", nil
	})

	shell.Register("server", func(args ...string) (string, error) {
		address := args[0]
		if strings.Contains(address, "http://") == false &&
			strings.Contains(address, "https://") == false {
			return "", errors.New("Address must be URL")
		}
		if server := c.findServer(address); server != nil {
			c.CurrentServer = server
			shell.Println("Changed current server to: ", server.Address)
			return "OK", nil
		}
		if err := c.addServer(address); err != nil {
			return "", err
		}
		c.StartHeartbeat()
		return "OK", nil
	})

	shell.Register("list-servers", func(args ...string) (string, error) {
		if len(c.Servers) == 0 {
			return "", errors.New("No servers")
		}
		for _, server := range c.Servers {
			if c.CurrentServer == server {
				fmt.Print("* ")
			} else {
				fmt.Print("| ")
			}
			fmt.Printf("Address: %s\t Missed Heartbeats: %d Online: %v\n",
				server.Address, server.MissedBeats, server.Online)
		}
		return "OK", nil
	})

	shell.Register("get-nodes", func(args ...string) (string, error) {
		if len(c.Servers) == 0 {
			return "", errors.New("No servers")
		}
		serial, err := c.CurrentServer.GetNodes(c.UUID, cliUserAgent)
		if err != nil {
			return "", err
		}
		if c.Config.OutputJSON == true {
			fmt.Println(string(serial))
			return "OK", nil
		}
		var nodes map[string]*api.Node
		if err := json.Unmarshal(serial, &nodes); err != nil {
			return "", err
		}
		c.prettyPrintNodes(nodes)
		return "OK", nil
	})
	shell.Register("sync-file", func(args ...string) (string, error) {
		if len(args) == 0 {
			return "", errors.New("Must specify file to sync")
		}
		fmt.Println("(Not yet implemented)")
		return "OK", nil
	})

	shell.Register("sync-dir", func(args ...string) (string, error) {
		if len(args) == 0 {
			return "", errors.New("Must specify dir to sync")
		}
		fmt.Println("(Not yet implemented)")
		return "OK", nil
	})

	shell.Register("get-index", func(args ...string) (string, error) {
		if len(args) == 0 {
			return "", errors.New("Must specify directory to index")
		}
		if len(c.Servers) == 0 {
			return "", errors.New("No servers")
		}
		dir := args[0]
		fmt.Println(dir)
		if dir == "" {
			return "", errors.New("Must specify directory")
		}
		serial, err := c.CurrentServer.RequestIndex(dir, c.UUID, cliUserAgent)
		if err != nil {
			return "", err
		}

		if c.Config.OutputJSON == true {
			fmt.Println(string(serial))
		} else {
			var remoteIndex map[string]*index.Index
			if err := json.Unmarshal(serial, &remoteIndex); err != nil {
				c.prettyPrintIndex(remoteIndex)
			}
		}
		return "", nil
	})
	shell.Start()
}
