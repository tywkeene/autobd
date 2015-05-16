package options

import (
	"flag"
	"fmt"
	"os"
)

type Options struct {
	Root    *string
	ApiPort *string
}

var Flags *Options

func GetOptions() {
	Flags = &Options{}
	Flags.Root = flag.String("root", "", "Root directory to serve (required)")
	Flags.ApiPort = flag.String("api-port", "8081", "Port that the API listens on")
	flag.Parse()

	if *Flags.Root == "" {
		fmt.Println("-root argument required")
		os.Exit(-1)
	}
}
