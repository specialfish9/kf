package main

import (
	"cmp"
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/logrusorgru/aurora/v4"
	"kf/config"
	"kf/kube"
	"kf/utils"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
)

func printFiglet() {
	fmt.Println(aurora.Bold(aurora.Green(" __      _____ ")))
	fmt.Println(aurora.Bold(aurora.Green("|  | ___/ ____\\ ")))
	fmt.Println(aurora.Bold(aurora.Green("|  |/ /\\   __\\ ")))
	fmt.Println(aurora.Bold(aurora.Green("|    <  |  |   ")))
	fmt.Println(aurora.Bold(aurora.Green("|__|_ \\ |__|   ")))
	fmt.Println(aurora.Bold(aurora.Green("     \\/       2")))
	fmt.Println(aurora.Bold(aurora.Green("")))
}

type opt struct {
	profile   *string
	service   *[]string
	forward   *[]string
	list      *bool
	namespace *string
	help      string
}

var serviceRx = regexp.MustCompile(`^([\w\-]+)(?::(\d{1,5}))?(?::(\d{1,5}))?$`)

func validateServiceArgs(args []string) error {
	for _, arg := range args {
		if !serviceRx.MatchString(arg) {
			return fmt.Errorf("invalid service format: %s", arg)
		}
	}
	return nil
}

func parseServiceArgs(args []string, mustHavePorts bool) []*config.Service {
	return utils.Map(args, func(s string) *config.Service {
		matches := serviceRx.FindStringSubmatch(s)
		if mustHavePorts && (matches[2] == "" || matches[3] == "") {
			log.Fatalf("invalid service format '%s'", s)
		}
		lp, _ := strconv.Atoi(matches[2])
		rp, _ := strconv.Atoi(matches[3])
		return &config.Service{
			Name:       matches[1],
			Alias:      matches[1],
			LocalPort:  lp,
			RemotePort: rp,
		}
	})
}

func parseArgs() *opt {
	opt := &opt{}
	parser := argparse.NewParser("kf", "")
	opt.profile = parser.String("p", "profile", &argparse.Options{Required: false, Help: "<profile_name> forward all services on the selected profile"})
	opt.service = parser.List("s", "service", &argparse.Options{Required: false, Help: "<alias>[:lport][:rport] ... forward one or more services from the config service list. lport/rport -> overrides the default port ", Validate: validateServiceArgs})
	opt.forward = parser.List("f", "forward", &argparse.Options{Required: false, Help: "<service_name><:lport><:rport> ... forward one or more services", Validate: validateServiceArgs})
	opt.list = parser.Flag("l", "list", &argparse.Options{Required: false, Help: "list all profiles and services"})
	opt.namespace = parser.String("n", "namespace", &argparse.Options{Required: false, Help: "kube namespace; defaults to dev; can be passed along with other args"})
	profile := parser.StringPositional(&argparse.Options{Required: false, Help: "forward all services on the selected profile; same as -p"})

	// Parse input
	err := parser.Parse(os.Args)
	parser.ExitOnHelp(true)
	if err != nil {
		log.Fatalf(parser.Usage(err))
	}
	opt.help = parser.Usage(nil)
	if *profile != "" {
		opt.profile = profile
	}
	return opt
}

func main() {
	printFiglet()
	opt := parseArgs()
	cfg, err := config.Read(config.DefaultPath())
	if err != nil {
		log.Fatalf("kf: unable to load config: %v", err.Error())
	}

	if *opt.profile != "" || len(*opt.service) > 0 || len(*opt.forward) > 0 {
		k, err := kube.NewKube()
		if err != nil {
			log.Fatalf("kf: error while connecting to kubernetes: %v", err.Error())
		}
		stopCh := make(chan struct{}, 1)

		if *opt.profile != "" {
			profile := cfg.GetProfile(*opt.profile)
			if profile == nil {
				log.Fatalf("kf: error spinning up services: unknown profile '%s'", *opt.profile)
			}
			k.ForwardProfile(profile, *opt.namespace, stopCh)
		} else if len(*opt.service) > 0 {
			services := parseServiceArgs(*opt.service, false)
			overlays := utils.Map(services, func(s *config.Service) *config.ServiceOverlay {
				ref := cfg.ServiceMap[s.Alias]
				if ref == nil {
					log.Fatalf("kf: error spinning up services: unknown service alias '%s'", s.Alias)
				}
				return &config.ServiceOverlay{
					Ref:        s.Alias,
					Service:    ref,
					LocalPort:  s.LocalPort,
					RemotePort: s.RemotePort,
				}
			})
			k.ForwardOverlays(overlays, cmp.Or(*opt.namespace, "dev"), stopCh)
		} else {
			services := parseServiceArgs(*opt.forward, true)
			k.ForwardServices(services, cmp.Or(*opt.namespace, "dev"), stopCh)
		}

		//waiting for interrupt
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
		if s := <-interrupt; true {
			log.Println("Signal: " + s.String())
			close(stopCh)
		}
		log.Println("Bye")
	} else if *opt.list {
		cfg.PrintList()
	} else {
		fmt.Print(opt.help)
	}
}
