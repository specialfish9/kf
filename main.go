package main

import (
	"cmp"
	"fmt"
	"kf/config"
	"kf/kube"
	"kf/utils"
	"log"
	"os"
	"os/signal"
	"syscall"
)

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
			k.ForwardOverlays(overlays, cmp.Or(*opt.namespace, config.DefaultEnv), stopCh)
		} else {
			services := parseServiceArgs(*opt.forward, true)
			k.ForwardServices(services, cmp.Or(*opt.namespace, config.DefaultEnv), stopCh)
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
