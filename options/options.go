package options

import (
	"flag"
	"fmt"
	"os"
)

type Options struct {
	Root    *string
	ApiPort *string
	Node    *bool
	Seed    *string
}

var Flags *Options

func GetOptions() {
	Flags = &Options{}
	Flags.Root = flag.String("root", "", "Root directory to serve (required)")
	Flags.ApiPort = flag.String("api-port", "8081", "Port that the API listens on")
	Flags.Node = flag.Bool("node", false, "Run as a node")
	Flags.Seed = flag.String("seed", "", "Seed server to connect with (required when running as a node)")
	flag.Parse()

	if *Flags.Root == "" {
		fmt.Println("-root argument required")
		os.Exit(-1)
	}

	if *Flags.Node == true && *Flags.Seed == "" {
		fmt.Println("Must specify seed server when running as node")
		os.Exit(-1)
	}
}
