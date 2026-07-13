package main

import (
	"fmt"
	"kf/config"
	"log"
	"os"

	"github.com/akamensky/argparse"
)

type opt struct {
	service   *string
	namespace *string
	verbose   *bool
	filter    *string
	all       *bool
	config    *string
	help      string
}

func parseArgs() *opt {
	opt := &opt{}
	parser := argparse.NewParser("kl", "Read those logs!")

	opt.config = parser.String("c", "config", &argparse.Options{
		Required: false,
		Help:     fmt.Sprintf("path to config file; defaults to %s", config.DefaultPath()),
	})

	opt.service = parser.StringPositional(&argparse.Options{
		Required: true,
		Help:     "<name|alias> specify the service",
	})

	opt.namespace = parser.String("n", "namespace", &argparse.Options{
		Required: false,
		Help:     "K8s namespace",
	})

	opt.filter = parser.String("f", "filter", &argparse.Options{
		Required: false,
		Help:     "regex filter",
	})

	opt.verbose = parser.Flag("v", "verbose", &argparse.Options{
		Required: false,
		Help:     "enable verbose logging",
	})

	opt.all = parser.Flag("a", "all", &argparse.Options{
		Required: false,
		Help:     "load all the logs since the creation of the pods",
	})

	// Parse input
	err := parser.Parse(os.Args)

	parser.ExitOnHelp(true)

	if err != nil {
		log.Fatalf(parser.Usage(err))
	}
	opt.help = parser.Usage(nil)

	return opt
}
